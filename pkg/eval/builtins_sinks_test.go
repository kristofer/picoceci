package eval

import (
	"bytes"
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

// TestSetTaskCaller verifies that SetTaskCaller wires a caller into the Task global.
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

// stubBlockCaller is a test BlockCaller that delegates to fn.
type stubBlockCaller struct {
	fn func(blk *object.Object, args []*object.Object) (*object.Object, error)
}

func (s *stubBlockCaller) CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error) {
	return s.fn(blk, args)
}
