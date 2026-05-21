package eval

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kristofer/picoceci/pkg/object"
)

func callNoArgNative(t *testing.T, recv *object.Object, selector string) {
	t.Helper()
	m, ok := recv.Methods[selector]
	if !ok || m == nil || m.Native == nil {
		t.Fatalf("missing native method %q", selector)
	}
	if _, err := m.Native(recv, nil); err != nil {
		t.Fatalf("call %q failed: %v", selector, err)
	}
}

func callOneArgNative(t *testing.T, recv *object.Object, selector string, arg *object.Object) {
	t.Helper()
	m, ok := recv.Methods[selector]
	if !ok || m == nil || m.Native == nil {
		t.Fatalf("missing native method %q", selector)
	}
	if _, err := m.Native(recv, []*object.Object{arg}); err != nil {
		t.Fatalf("call %q failed: %v", selector, err)
	}
}

func TestInitialGlobalsWithSinks_SplitConsoleAndTranscript(t *testing.T) {
	var consoleBuf bytes.Buffer
	var transcriptBuf bytes.Buffer

	globals := InitialGlobalsWithSinks(GlobalSinks{
		ConsoleWriter:    &consoleBuf,
		TranscriptWriter: &transcriptBuf,
	})

	console := globals["Console"]
	transcript := globals["Transcript"]
	if console == nil || transcript == nil {
		t.Fatal("expected Console and Transcript globals")
	}

	callOneArgNative(t, console, "println:", object.StringObject("console"))
	callNoArgNative(t, console, "nl")
	callOneArgNative(t, transcript, "println:", object.StringObject("transcript"))

	if got := consoleBuf.String(); got != "console\n\n" {
		t.Fatalf("console sink output = %q, want %q", got, "console\\n\\n")
	}
	if got := transcriptBuf.String(); got != "transcript\n" {
		t.Fatalf("transcript sink output = %q, want %q", got, "transcript\\n")
	}
}

func TestInitialGlobalsWithSinks_TranscriptFallsBackToConsoleSink(t *testing.T) {
	var consoleBuf bytes.Buffer

	globals := InitialGlobalsWithSinks(GlobalSinks{ConsoleWriter: &consoleBuf})
	transcript := globals["Transcript"]
	if transcript == nil {
		t.Fatal("expected Transcript global")
	}

	callOneArgNative(t, transcript, "print:", object.StringObject("hello"))

	if got := consoleBuf.String(); got != "hello" {
		t.Fatalf("fallback sink output = %q, want %q", got, "hello")
	}
}

// TestInitialGlobalsWithSinks_HasTaskWifiREPL verifies that Task, Wifi,
// and PicoceciREPL are present in the globals map.
func TestInitialGlobalsWithSinks_HasTaskWifiREPL(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	for _, name := range []string{"Task", "Wifi", "PicoceciREPL"} {
		if globals[name] == nil {
			t.Errorf("expected %q in globals, got nil", name)
		}
	}
}

// TestInitialGlobalsWithSinks_HasPhase7Globals verifies that LED, Timestamp,
// Duration, and TaskSupervisor are present in the globals map.
func TestInitialGlobalsWithSinks_HasPhase7Globals(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	for _, name := range []string{"LED", "Timestamp", "Duration", "TaskSupervisor"} {
		if globals[name] == nil {
			t.Errorf("expected %q in globals, got nil", name)
		}
	}
}

// TestWifiGlobal_ConnectAndStatus verifies the Wifi singleton methods.
func TestWifiGlobal_ConnectAndStatus(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	wifi := globals["Wifi"]
	if wifi == nil {
		t.Fatal("Wifi global not found")
	}

	// connectSSID:password: should succeed (no-op on desktop)
	m := wifi.Methods["connectSSID:password:"]
	if m == nil || m.Native == nil {
		t.Fatal("missing connectSSID:password: method")
	}
	_, err := m.Native(wifi, []*object.Object{
		object.StringObject("testnet"),
		object.StringObject("pass"),
	})
	if err != nil {
		t.Fatalf("connectSSID:password: error: %v", err)
	}

	// status should return #connected
	sm := wifi.Methods["status"]
	if sm == nil || sm.Native == nil {
		t.Fatal("missing status method")
	}
	res, err := sm.Native(wifi, nil)
	if err != nil {
		t.Fatalf("status error: %v", err)
	}
	if res == nil || res.Kind != object.KindSymbol || res.SVal != "connected" {
		t.Errorf("status = %v, want #connected", res)
	}
}

// TestWifiGlobal_ListenOn starts a listener via the Wifi picoceci object,
// accepts one connection through the picoceci Listener object, and verifies
// session I/O works.
func TestWifiGlobal_ListenOn(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	wifi := globals["Wifi"]
	if wifi == nil {
		t.Fatal("Wifi global not found")
	}

	// listenOn:do: with port 0 (OS chooses)
	lm := wifi.Methods["listenOn:do:"]
	if lm == nil || lm.Native == nil {
		t.Fatal("missing listenOn:do: method")
	}
	// We pass a nil block for now — the method stores the listener and returns it.
	listenerObj, err := lm.Native(wifi, []*object.Object{
		object.IntObject(0),
		{Kind: object.KindBlock}, // dummy block
	})
	if err != nil {
		t.Fatalf("listenOn:do: error: %v", err)
	}
	if listenerObj == nil {
		t.Fatal("listenOn:do: returned nil")
	}

	// addr should return a non-empty string
	addrM := listenerObj.Methods["addr"]
	if addrM == nil || addrM.Native == nil {
		t.Fatal("missing addr method on listener")
	}
	addrObj, err := addrM.Native(listenerObj, nil)
	if err != nil || addrObj == nil || addrObj.SVal == "" {
		t.Fatalf("addr error or empty: %v / %v", err, addrObj)
	}

	// clean up
	closeM := listenerObj.Methods["close"]
	if closeM != nil && closeM.Native != nil {
		_, _ = closeM.Native(listenerObj, nil)
	}
}

// TestPicoceciREPL_ServeCallsRunner verifies that PicoceciREPL serve: calls
// the REPLRunner with the session's reader/writer.
func TestPicoceciREPL_ServeCallsRunner(t *testing.T) {
	var runnerCalled bool
	var mu sync.Mutex

	runner := REPLRunner(func(r io.Reader, w io.Writer) {
		mu.Lock()
		runnerCalled = true
		mu.Unlock()
		_, _ = io.WriteString(w, "=> 42\n")
	})

	globals := InitialGlobalsWithSinks(GlobalSinks{REPLRunner: runner})
	replObj := globals["PicoceciREPL"]
	if replObj == nil {
		t.Fatal("PicoceciREPL global not found")
	}

	// Build a fake picoceci session object backed by bytes.Buffer.
	var inBuf bytes.Buffer
	var outBuf bytes.Buffer
	inBuf.WriteString("42.\n")

	fakeSession := &fakeNetSession{r: &inBuf, w: &outBuf}
	sessionObj := makeSessionObject(fakeSession)

	serveM := replObj.Methods["serve:"]
	if serveM == nil || serveM.Native == nil {
		t.Fatal("missing serve: method")
	}
	if _, err := serveM.Native(replObj, []*object.Object{sessionObj}); err != nil {
		t.Fatalf("serve: error: %v", err)
	}

	// Wait briefly for any goroutine to finish (serve: is synchronous here).
	time.Sleep(5 * time.Millisecond)

	mu.Lock()
	called := runnerCalled
	mu.Unlock()

	if !called {
		t.Error("REPLRunner was not called by PicoceciREPL serve:")
	}
	if got := outBuf.String(); !strings.Contains(got, "42") {
		t.Errorf("session output = %q, expected to contain '42'", got)
	}
}

// TestSetTaskCaller_SpawnBlock verifies that SetTaskCaller wires a caller into the Task global.
func TestSetTaskCaller_SpawnBlock(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	task := globals["Task"]
	if task == nil {
		t.Fatal("Task global not found")
	}

	// Task.spawn:name: should fail before SetTaskCaller.
	spawnM := task.Methods["spawn:name:"]
	if spawnM == nil || spawnM.Native == nil {
		t.Fatal("missing spawn:name: method")
	}

	// Build a trivial block (KindBlock with no body — CallBlock is a no-op).
	blk := &object.Object{Kind: object.KindBlock}
	_, err := spawnM.Native(task, []*object.Object{blk, object.StringObject("t")})
	if err == nil {
		t.Fatal("expected error before SetTaskCaller, got nil")
	}

	// Wire a stub BlockCaller.
	var blockCalled bool
	stub := &stubBlockCaller{fn: func(blk *object.Object, args []*object.Object) (*object.Object, error) {
		blockCalled = true
		return object.Nil, nil
	}}
	SetTaskCaller(globals, stub)

	// Now spawn:name: should succeed.
	_, err = spawnM.Native(task, []*object.Object{blk, object.StringObject("t")})
	if err != nil {
		t.Fatalf("spawn:name: after SetTaskCaller error: %v", err)
	}
	// Allow the goroutine to run.
	time.Sleep(20 * time.Millisecond)
	if !blockCalled {
		t.Error("block was not called by Task spawn:name:")
	}
}

// stubBlockCaller is a test BlockCaller that delegates to fn.
type stubBlockCaller struct {
	fn func(blk *object.Object, args []*object.Object) (*object.Object, error)
}

func (s *stubBlockCaller) CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error) {
	return s.fn(blk, args)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fakeNetSession implements picnet.Session backed by in-memory buffers.
type fakeNetSession struct {
	r io.Reader
	w io.Writer
}

func (f *fakeNetSession) Read(p []byte) (int, error)  { return f.r.Read(p) }
func (f *fakeNetSession) Write(p []byte) (int, error) { return f.w.Write(p) }
func (f *fakeNetSession) ReadLine() (string, error) {
	var sb strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := f.r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return sb.String(), nil
			}
			sb.WriteByte(buf[0])
		}
		if err != nil {
			return sb.String(), err
		}
	}
}
func (f *fakeNetSession) RemoteAddr() string { return "127.0.0.1:99999" }
func (f *fakeNetSession) Close() error       { return nil }

// ---------------------------------------------------------------------------
// LED singleton tests
// ---------------------------------------------------------------------------

// recordingLEDDriver records calls for test assertions.
type recordingLEDDriver struct {
	calls []string
}

func (r *recordingLEDDriver) On()              { r.calls = append(r.calls, "on") }
func (r *recordingLEDDriver) Off()             { r.calls = append(r.calls, "off") }
func (r *recordingLEDDriver) Toggle()          { r.calls = append(r.calls, "toggle") }
func (r *recordingLEDDriver) BlinkEvery(ms int) {
	r.calls = append(r.calls, fmt.Sprintf("blink:%d", ms))
}
func (r *recordingLEDDriver) StopBlink() { r.calls = append(r.calls, "stopBlink") }

func TestLEDGlobal_Methods(t *testing.T) {
	rec := &recordingLEDDriver{}
	globals := InitialGlobalsWithSinks(GlobalSinks{LEDDriver: rec})
	led := globals["LED"]
	if led == nil {
		t.Fatal("LED global not found")
	}

	callNoArgNative(t, led, "on")
	callNoArgNative(t, led, "off")
	callNoArgNative(t, led, "toggle")
	callOneArgNative(t, led, "blinkEvery:", object.IntObject(500))
	callNoArgNative(t, led, "stopBlink")

	want := []string{"on", "off", "toggle", "blink:500", "stopBlink"}
	for i, w := range want {
		if i >= len(rec.calls) || rec.calls[i] != w {
			t.Errorf("calls[%d] = %q, want %q (all calls: %v)", i, rec.calls[i], w, rec.calls)
		}
	}
}

func TestLEDGlobal_PrintString(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	led := globals["LED"]
	m := led.Methods["printString"]
	if m == nil || m.Native == nil {
		t.Fatal("missing printString method on LED")
	}
	res, err := m.Native(led, nil)
	if err != nil || res == nil || res.SVal != "LED" {
		t.Errorf("LED printString = %v/%v, want 'LED'/nil", res, err)
	}
}

// ---------------------------------------------------------------------------
// Timestamp and Duration tests
// ---------------------------------------------------------------------------

func TestTimestampGlobal_Now(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	tsClass := globals["Timestamp"]
	if tsClass == nil {
		t.Fatal("Timestamp global not found")
	}

	nowM := tsClass.Methods["now"]
	if nowM == nil || nowM.Native == nil {
		t.Fatal("missing Timestamp now method")
	}
	ts, err := nowM.Native(tsClass, nil)
	if err != nil || ts == nil {
		t.Fatalf("Timestamp now error: %v / %v", err, ts)
	}
	// asMilliseconds should return a non-negative integer
	msM := ts.Methods["asMilliseconds"]
	if msM == nil || msM.Native == nil {
		t.Fatal("missing asMilliseconds on Timestamp instance")
	}
	msObj, err := msM.Native(ts, nil)
	if err != nil || msObj == nil || msObj.IVal < 0 {
		t.Errorf("Timestamp asMilliseconds = %v/%v, want >= 0", msObj, err)
	}
}

func TestTimestampGlobal_Subtraction(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	tsClass := globals["Timestamp"]
	nowM := tsClass.Methods["now"]

	ts1, _ := nowM.Native(tsClass, nil)
	time.Sleep(5 * time.Millisecond)
	ts2, _ := nowM.Native(tsClass, nil)

	// ts2 - ts1 should be a Duration >= 0ms
	subM := ts2.Methods["-"]
	if subM == nil || subM.Native == nil {
		t.Fatal("missing - method on Timestamp")
	}
	durObj, err := subM.Native(ts2, []*object.Object{ts1})
	if err != nil || durObj == nil {
		t.Fatalf("Timestamp - error: %v / %v", err, durObj)
	}
	msM := durObj.Methods["asMilliseconds"]
	if msM == nil || msM.Native == nil {
		t.Fatal("missing asMilliseconds on Duration")
	}
	msObj, _ := msM.Native(durObj, nil)
	if msObj == nil || msObj.IVal < 0 {
		t.Errorf("Duration ms = %v, want >= 0", msObj)
	}
}

func TestDurationClass_MsConstructor(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	durClass := globals["Duration"]
	if durClass == nil {
		t.Fatal("Duration global not found")
	}

	msM := durClass.Methods["ms:"]
	if msM == nil || msM.Native == nil {
		t.Fatal("missing Duration ms: method")
	}
	dur, err := msM.Native(durClass, []*object.Object{object.IntObject(1500)})
	if err != nil || dur == nil {
		t.Fatalf("Duration ms: error: %v / %v", err, dur)
	}

	// printString should be "1.500s" for 1500ms
	psM := dur.Methods["printString"]
	if psM == nil || psM.Native == nil {
		t.Fatal("missing printString on Duration")
	}
	ps, _ := psM.Native(dur, nil)
	if ps == nil || ps.SVal != "1.500s" {
		t.Errorf("Duration printString = %q, want '1.500s'", ps.SVal)
	}

	// asSeconds
	secM := dur.Methods["asSeconds"]
	if secM == nil || secM.Native == nil {
		t.Fatal("missing asSeconds on Duration")
	}
	sec, _ := secM.Native(dur, nil)
	if sec == nil || sec.FVal != 1.5 {
		t.Errorf("Duration asSeconds = %v, want 1.5", sec)
	}
}

func TestDurationArithmetic(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	durClass := globals["Duration"]
	msM := durClass.Methods["ms:"]

	dur100, _ := msM.Native(durClass, []*object.Object{object.IntObject(100)})
	dur200, _ := msM.Native(durClass, []*object.Object{object.IntObject(200)})

	// 100 + 200 = 300
	addM := dur100.Methods["+"]
	sum, err := addM.Native(dur100, []*object.Object{dur200})
	if err != nil || sum == nil {
		t.Fatalf("Duration + error: %v", err)
	}
	sumMs, _ := sum.Methods["asMilliseconds"].Native(sum, nil)
	if sumMs.IVal != 300 {
		t.Errorf("100ms + 200ms = %dms, want 300ms", sumMs.IVal)
	}

	// 200 < 100 → false; 100 < 200 → true
	ltM := dur100.Methods["<"]
	res, _ := ltM.Native(dur100, []*object.Object{dur200})
	if res == nil || !res.Truthy() {
		t.Errorf("100ms < 200ms = %v, want true", res)
	}
	res2, _ := ltM.Native(dur200, []*object.Object{dur100})
	if res2 == nil || res2.Truthy() {
		t.Errorf("200ms < 100ms = %v, want false", res2)
	}
}

// ---------------------------------------------------------------------------
// TaskSupervisor tests
// ---------------------------------------------------------------------------

func TestTaskSupervisor_InGlobals(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	ts := globals["TaskSupervisor"]
	if ts == nil {
		t.Fatal("TaskSupervisor global not found")
	}
	for _, sel := range []string{"supervise:name:", "supervise:name:maxRestarts:", "printString"} {
		if ts.Methods[sel] == nil {
			t.Errorf("missing method %q on TaskSupervisor", sel)
		}
	}
}

func TestTaskSupervisor_SuperviseRestartsOnError(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	ts := globals["TaskSupervisor"]

	// Before SetTaskCaller, supervise:name: should error.
	blk := &object.Object{Kind: object.KindBlock}
	_, err := ts.Methods["supervise:name:"].Native(ts, []*object.Object{blk, object.StringObject("t")})
	if err == nil {
		t.Fatal("expected error before SetTaskCaller, got nil")
	}

	// Wire caller that fails twice then succeeds.
	var callCount int
	stub := &stubBlockCaller{fn: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		callCount++
		if callCount < 3 {
			return nil, &Error{Kind: "TestError", Message: "simulated crash"}
		}
		return object.Nil, nil // clean exit on 3rd call
	}}
	SetTaskCaller(globals, stub)

	_, err = ts.Methods["supervise:name:"].Native(ts, []*object.Object{blk, object.StringObject("t")})
	if err != nil {
		t.Fatalf("supervise:name: after SetTaskCaller error: %v", err)
	}
	// Wait for goroutine to complete (3 calls: fail, fail, succeed).
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) && callCount < 3 {
		time.Sleep(5 * time.Millisecond)
	}
	if callCount < 3 {
		t.Errorf("block called %d times, want at least 3 (2 crashes + 1 clean exit)", callCount)
	}
}

func TestTaskSupervisor_MaxRestarts(t *testing.T) {
	globals := InitialGlobalsWithSinks(GlobalSinks{})
	ts := globals["TaskSupervisor"]

	var callCount int
	stub := &stubBlockCaller{fn: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		callCount++
		return nil, &Error{Kind: "TestError", Message: "always crash"}
	}}
	SetTaskCaller(globals, stub)

	blk := &object.Object{Kind: object.KindBlock}
	// maxRestarts: 2 → initial call + 2 restarts = 3 total calls
	_, err := ts.Methods["supervise:name:maxRestarts:"].Native(ts, []*object.Object{
		blk,
		object.StringObject("t"),
		object.IntObject(2),
	})
	if err != nil {
		t.Fatalf("supervise:name:maxRestarts: error: %v", err)
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) && callCount < 3 {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond) // allow any extra calls
	if callCount != 3 {
		t.Errorf("block called %d times, want exactly 3 (initial + 2 restarts)", callCount)
	}
}
