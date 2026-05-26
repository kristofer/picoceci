package eval

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/freertos"
	picnet "github.com/kristofer/picoceci/pkg/net"
	"github.com/kristofer/picoceci/pkg/object"
	"github.com/kristofer/picoceci/pkg/sdcard"
)

// BlockCaller is an interface that allows builtins to invoke blocks.
// Both the tree-walking interpreter and bytecode VM implement this.
type BlockCaller interface {
	CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error)
}

// SourceRunner is an interface that allows builtins to parse and evaluate
// picoceci source code in the current global environment.
// Both the tree-walking interpreter and bytecode VM implement this.
type SourceRunner interface {
	EvalSource(source string) (*object.Object, error)
}

// REPLRunner is a function that runs a REPL session on the given reader/writer.
// It is called by PicoceciREPL serve: when a client connects.
// The function should block until the session ends (client disconnects or EOF).
type REPLRunner func(r io.Reader, w io.Writer)

// LEDDriver is the platform-provided LED controller.
// Implement this interface to wire a real LED into the LED singleton.
// A no-op stub is used when GlobalSinks.LEDDriver is nil.
type LEDDriver interface {
	On()
	Off()
	Toggle()
	Red()
	Green()
	Blue()
	White()
	RGB(r, g, b uint8)
	BlinkEvery(ms int)
	StopBlink()
}

// noopLEDDriver is a silent LED stub used on desktop when no driver is injected.
type noopLEDDriver struct{}

func (noopLEDDriver) On()               {}
func (noopLEDDriver) Off()              {}
func (noopLEDDriver) Toggle()           {}
func (noopLEDDriver) Red()              {}
func (noopLEDDriver) Green()            {}
func (noopLEDDriver) Blue()             {}
func (noopLEDDriver) White()            {}
func (noopLEDDriver) RGB(_, _, _ uint8) {}
func (noopLEDDriver) BlinkEvery(_ int)  {}
func (noopLEDDriver) StopBlink()        {}

// GlobalSinks configures output destinations for built-in global objects.
// Console and Transcript can be routed independently.
// WifiManager and REPLRunner wire WiFi and remote-REPL support.
type GlobalSinks struct {
	ConsoleWriter    io.Writer
	TranscriptWriter io.Writer

	// WifiManager, when non-nil, backs the Wifi singleton.
	// If nil, a new default Manager is created automatically.
	WifiManager *picnet.Manager

	// REPLRunner, when non-nil, is called by PicoceciREPL serve: session.
	// It receives the session's Reader and Writer and should run until the
	// session ends.  If nil, PicoceciREPL.serve: is a no-op.
	REPLRunner REPLRunner

	// LEDDriver, when non-nil, backs the LED singleton with real hardware.
	// If nil, a no-op stub is used.
	LEDDriver LEDDriver
}

// InitialGlobals returns a map of global names to their initial values.
// This includes: nil, true, false (picoceci keywords), Console, Transcript,
// Array, Queue, Channel, Task, Wifi, PicoceciREPL, LED, Timestamp, Duration,
// TaskSupervisor.
// Both the tree-walking interpreter and bytecode VM use this.
func InitialGlobals() map[string]*object.Object {
	return InitialGlobalsWithSinks(GlobalSinks{})
}

// InitialGlobalsWithSinks returns initial globals with configurable output sinks.
func InitialGlobalsWithSinks(sinks GlobalSinks) map[string]*object.Object {
	globals := make(map[string]*object.Object)

	globals["nil"] = object.Nil
	globals["true"] = object.True
	globals["false"] = object.False

	// Console / Transcript
	console := makeConsole(sinks.ConsoleWriter)
	transcriptWriter := sinks.TranscriptWriter
	if transcriptWriter == nil {
		transcriptWriter = sinks.ConsoleWriter
	}
	transcript := makeConsole(transcriptWriter)
	globals["Console"] = console
	globals["Transcript"] = transcript

	// Array class object
	globals["Array"] = makeArrayClass()
	globals["Queue"] = makeQueueClass()
	globals["Channel"] = makeChannelClass()

	// Task singleton — spawns picoceci blocks as goroutines.
	// The BlockCaller is set lazily via SetTaskCaller after interpreter/VM init.
	taskData := &taskObjectData{}
	globals["Task"] = makeTaskObject(taskData)
	globals["TaskSupervisor"] = makeTaskSupervisorObject(taskData)

	// Wifi singleton — WiFi station management + TCP listener setup.
	wifiMgr := sinks.WifiManager
	if wifiMgr == nil {
		wifiMgr = picnet.NewManager()
	}
	globals["Wifi"] = makeWifiObject(wifiMgr, globals)

	// PicoceciREPL singleton — runs a REPL on a network session.
	globals["PicoceciREPL"] = makePicoceciREPLObject(sinks.REPLRunner)

	// LED singleton — board LED status and heartbeat control.
	ledDriver := sinks.LEDDriver
	if ledDriver == nil {
		ledDriver = noopLEDDriver{}
	}
	globals["LED"] = makeLEDObject(ledDriver)
	globals["SDCard"] = makeSDCardObject()
	// File singleton — file operations with runContents: for script loading.
	// The SourceRunner is set lazily via SetFileRunner after interpreter/VM init.
	fileData := &fileObjectData{}
	globals["File"] = makeFileClass(fileData)
	globals["Directory"] = makeDirectoryClass()
	globals["Path"] = makePathClass()

	// Timestamp class — milliseconds since boot.
	globals["Timestamp"] = makeTimestampClass()

	// Duration class — millisecond-based time spans.
	globals["Duration"] = makeDurationClass()

	return globals
}

// SetTaskCaller wires the BlockCaller (interpreter or VM) into the Task global
// so that Task spawn:name: can call picoceci blocks.
// Call this once after creating the interpreter or VM.
func SetTaskCaller(globals map[string]*object.Object, caller BlockCaller) {
	if taskObj, ok := globals["Task"]; ok {
		if data, ok := taskObj.Env.(*taskObjectData); ok {
			data.caller = caller
		}
	}
}

// SetFileRunner wires the SourceRunner (interpreter or VM) into the File global
// so that File runContents: can parse and evaluate picoceci source files.
// Call this once after creating the interpreter or VM.
func SetFileRunner(globals map[string]*object.Object, runner SourceRunner) {
	if fileObj, ok := globals["File"]; ok {
		if data, ok := fileObj.Env.(*fileObjectData); ok {
			data.runner = runner
		}
	}
}

// registerBuiltins populates the global environment with built-in objects.
// This is used by the tree-walking interpreter.
func registerBuiltins(env *Env) {
	registerBuiltinsWithGlobals(env, InitialGlobals())
}

func registerBuiltinsWithGlobals(env *Env, globals map[string]*object.Object) {
	for name, val := range globals {
		env.Define(name)
		env.Set(name, val)
	}
}

func displayString(o *object.Object) string {
	if o == nil {
		return "nil"
	}
	// For strings and symbols, return raw value (no quotes) — like displayString.
	if o.Kind == object.KindString {
		return o.SVal
	}
	if o.Kind == object.KindSymbol {
		return o.SVal
	}
	return o.PrintString()
}

func makeConsole(writer io.Writer) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	printSink := func(s string) {
		if writer != nil {
			_, _ = io.WriteString(writer, s)
			return
		}
		fmt.Print(s)
	}
	printlnSink := func(s string) {
		if writer != nil {
			_, _ = io.WriteString(writer, s+"\n")
			return
		}
		fmt.Println(s)
	}
	o.Methods["print:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) > 0 {
			printSink(displayString(args[0]))
		}
		return object.Nil, nil
	}}
	o.Methods["println:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) > 0 {
			printlnSink(displayString(args[0]))
		}
		return object.Nil, nil
	}}
	o.Methods["show:"] = o.Methods["println:"]
	o.Methods["nl"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		printlnSink("")
		return object.Nil, nil
	}}
	return o
}

func makeArrayClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	o.Methods["new:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) > 0 && args[0].Kind == object.KindSmallInt {
			n := args[0].IVal
			if n < 0 || n > math.MaxInt {
				return object.ArrayObject(0), nil
			}
			return object.ArrayObject(int(n)), nil
		}
		return object.ArrayObject(0), nil
	}}
	o.Methods["new:withAll:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) == 2 && args[0].Kind == object.KindSmallInt {
			n := args[0].IVal
			if n < 0 || n > math.MaxInt {
				return object.ArrayObject(0), nil
			}
			size := int(n)
			arr := &object.Object{Kind: object.KindArray, Items: make([]*object.Object, size)}
			for i := range arr.Items {
				arr.Items[i] = args[1]
			}
			return arr, nil
		}
		return object.ArrayObject(0), nil
	}}
	return o
}

type queueObjectData struct {
	queue    freertos.Queue
	kindName string
}

func makeQueueClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	o.Methods["new:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		return newQueueLikeInstance("Queue", args)
	}}
	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Queue"), nil
	}}
	return o
}

func makeChannelClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	o.Methods["new:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		return newQueueLikeInstance("Channel", args)
	}}
	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Channel"), nil
	}}
	return o
}

func newQueueLikeInstance(kindName string, args []*object.Object) (*object.Object, error) {
	capacity := 0
	if len(args) > 0 && args[0] != nil && args[0].Kind == object.KindSmallInt {
		if args[0].IVal > 0 && args[0].IVal <= math.MaxInt {
			capacity = int(args[0].IVal)
		}
	}

	inst := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	inst.Env = &queueObjectData{
		queue:    freertos.NewQueue(capacity),
		kindName: kindName,
	}

	inst.Methods["receive"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		data, err := queueData(self)
		if err != nil {
			return nil, err
		}
		// Use timeout 0 for host-friendly polling semantics in tests and REPL use.
		item, ok := data.queue.Receive(0)
		if !ok || item == nil {
			return object.Nil, nil
		}
		if obj, ok := item.(*object.Object); ok {
			return obj, nil
		}
		return object.Nil, nil
	}}
	inst.Methods["send:"] = &object.MethodDef{Native: func(self *object.Object, msgArgs []*object.Object) (*object.Object, error) {
		if len(msgArgs) == 0 {
			return object.Nil, nil
		}
		data, err := queueData(self)
		if err != nil {
			return nil, err
		}
		if !data.queue.Send(msgArgs[0], 0) {
			return nil, &Error{Kind: "TaskError", Message: data.kindName + " send failed", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return msgArgs[0], nil
	}}
	inst.Methods["count"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		data, err := queueData(self)
		if err != nil {
			return nil, err
		}
		return object.IntObject(int64(data.queue.Count())), nil
	}}
	inst.Methods["printString"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		data, err := queueData(self)
		if err != nil {
			return nil, err
		}
		return object.StringObject("a " + data.kindName), nil
	}}

	if kindName == "Channel" {
		inst.Methods["<-"] = inst.Methods["send:"]
	}

	return inst, nil
}

func queueData(self *object.Object) (*queueObjectData, error) {
	if self == nil {
		return nil, &Error{Kind: "TaskError", Message: "queue object is nil", Pos: ast.Pos{Line: 1, Col: 1}}
	}
	data, ok := self.Env.(*queueObjectData)
	if !ok || data == nil || data.queue == nil {
		return nil, &Error{Kind: "TaskError", Message: "queue object is not initialized", Pos: ast.Pos{Line: 1, Col: 1}}
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Task global
// ---------------------------------------------------------------------------

// taskObjectData holds the lazy BlockCaller for the Task singleton.
type taskObjectData struct {
	caller BlockCaller
}

// makeTaskObject creates the Task singleton picoceci object.
// data.caller must be set via SetTaskCaller before Task spawn:name: is usable.
func makeTaskObject(data *taskObjectData) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     data,
	}

	// Task spawn: aBlock name: aString
	// Spawns aBlock in a new goroutine.  Returns the task name as a Symbol.
	o.Methods["spawn:name:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td, ok := self.Env.(*taskObjectData)
		if !ok || td == nil || td.caller == nil {
			return nil, &Error{Kind: "TaskError", Message: "Task not initialized; call SetTaskCaller after creating interpreter", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		if len(args) < 2 {
			return object.Nil, nil
		}
		blk := args[0]
		if blk == nil || blk.Kind != object.KindBlock {
			return nil, &Error{Kind: "TaskError", Message: "Task spawn:name: first argument must be a Block", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		name := ""
		if args[1] != nil {
			name = args[1].SVal
		}
		caller := td.caller
		go func() {
			_, _ = caller.CallBlock(blk, nil)
		}()
		return object.SymbolObject(name), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Task"), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// Wifi global
// ---------------------------------------------------------------------------

// wifiObjectData holds the WiFi manager and the globals map used to rebind
// Transcript when a session connects.
type wifiObjectData struct {
	mgr     *picnet.Manager
	globals map[string]*object.Object // reference to globals for Transcript rebind
}

// makeWifiObject creates the Wifi singleton picoceci object.
// globals is kept as a reference so listenOn:do: can rebind Transcript.
func makeWifiObject(mgr *picnet.Manager, globals map[string]*object.Object) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     &wifiObjectData{mgr: mgr, globals: globals},
	}

	// Wifi connectSSID: ssid password: pass
	o.Methods["connectSSID:password:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		wd := self.Env.(*wifiObjectData)
		ssid := ""
		pass := ""
		if len(args) > 0 && args[0] != nil {
			ssid = args[0].SVal
		}
		if len(args) > 1 && args[1] != nil {
			pass = args[1].SVal
		}
		if err := wd.mgr.Connect(ssid, pass); err != nil {
			return nil, &Error{Kind: "IOError", Message: "Wifi connect: " + err.Error(), Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return object.Nil, nil
	}}

	// Wifi status  → #idle | #connecting | #connected | #error
	o.Methods["status"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		wd := self.Env.(*wifiObjectData)
		return object.SymbolObject(wd.mgr.Status().String()), nil
	}}

	// Wifi ipAddress  → String
	o.Methods["ipAddress"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		wd := self.Env.(*wifiObjectData)
		ip := wd.mgr.IPAddress()
		if ip == "" {
			return object.Nil, nil
		}
		return object.StringObject(ip), nil
	}}

	// Wifi listenOn: port do: aBlock
	// Opens a TCP listener on port, then for each accepted session calls aBlock
	// with the session picoceci object as the argument.
	// The block is called in the current goroutine (blocking); wrap in Task spawn:
	// to run concurrently.
	o.Methods["listenOn:do:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		wd := self.Env.(*wifiObjectData)
		if len(args) < 2 {
			return object.Nil, nil
		}
		portObj := args[0]
		blk := args[1]
		if portObj == nil || portObj.Kind != object.KindSmallInt {
			return nil, &Error{Kind: "IOError", Message: "Wifi listenOn:do: port must be an Integer", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		if blk == nil || blk.Kind != object.KindBlock {
			return nil, &Error{Kind: "IOError", Message: "Wifi listenOn:do: second argument must be a Block", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		ln, err := wd.mgr.Listen(int(portObj.IVal))
		if err != nil {
			return nil, &Error{Kind: "IOError", Message: "Wifi listenOn: " + err.Error(), Pos: ast.Pos{Line: 1, Col: 1}}
		}
		// Store the listener so callers can close it later.
		self.Slots["_listener"] = makeListenerObject(ln)
		return self.Slots["_listener"], nil
	}}

	// Wifi disconnect
	o.Methods["disconnect"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		wd := self.Env.(*wifiObjectData)
		if err := wd.mgr.Disconnect(); err != nil {
			return nil, &Error{Kind: "IOError", Message: "Wifi disconnect: " + err.Error(), Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return object.Nil, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Wifi"), nil
	}}

	return o
}

// makeListenerObject wraps a net.Listener as a picoceci object.
func makeListenerObject(ln picnet.Listener) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     ln,
	}

	o.Methods["accept"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		l, ok := self.Env.(picnet.Listener)
		if !ok || l == nil {
			return nil, &Error{Kind: "IOError", Message: "listener not initialized", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		sess, err := l.Accept()
		if err != nil {
			return nil, &Error{Kind: "IOError", Message: "listener accept: " + err.Error(), Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return makeSessionObject(sess), nil
	}}

	o.Methods["close"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		l, ok := self.Env.(picnet.Listener)
		if !ok || l == nil {
			return object.Nil, nil
		}
		_ = l.Close()
		return object.Nil, nil
	}}

	o.Methods["addr"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		l, ok := self.Env.(picnet.Listener)
		if !ok || l == nil {
			return object.Nil, nil
		}
		return object.StringObject(l.Addr()), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("a TCPListener"), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// Session picoceci object
// ---------------------------------------------------------------------------

// makeSessionObject wraps a net.Session as a picoceci object.
// The session exposes readLine, write:, remoteAddr, close, and asWriter.
func makeSessionObject(sess picnet.Session) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     sess,
	}

	o.Methods["readLine"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		s, ok := self.Env.(picnet.Session)
		if !ok || s == nil {
			return object.Nil, nil
		}
		line, err := s.ReadLine()
		if err != nil {
			return object.Nil, nil
		}
		return object.StringObject(line), nil
	}}

	o.Methods["write:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		s, ok := self.Env.(picnet.Session)
		if !ok || s == nil {
			return object.Nil, nil
		}
		if len(args) > 0 && args[0] != nil {
			_, _ = io.WriteString(s, displayString(args[0]))
		}
		return object.Nil, nil
	}}

	o.Methods["writeln:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		s, ok := self.Env.(picnet.Session)
		if !ok || s == nil {
			return object.Nil, nil
		}
		if len(args) > 0 && args[0] != nil {
			_, _ = io.WriteString(s, displayString(args[0])+"\n")
		}
		return object.Nil, nil
	}}

	o.Methods["remoteAddr"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		s, ok := self.Env.(picnet.Session)
		if !ok || s == nil {
			return object.Nil, nil
		}
		return object.StringObject(s.RemoteAddr()), nil
	}}

	o.Methods["close"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		s, ok := self.Env.(picnet.Session)
		if !ok || s == nil {
			return object.Nil, nil
		}
		_ = s.Close()
		return object.Nil, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("a TCPSession"), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// PicoceciREPL global
// ---------------------------------------------------------------------------

// makePicoceciREPLObject creates the PicoceciREPL singleton.
// runner is called for each session; if nil serve: is a no-op.
func makePicoceciREPLObject(runner REPLRunner) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     runner,
	}

	// PicoceciREPL serve: session
	// Runs the REPL runner on the session.  Blocks until the session ends.
	o.Methods["serve:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		fn, _ := self.Env.(REPLRunner)
		if fn == nil {
			return object.Nil, nil
		}
		if len(args) == 0 || args[0] == nil {
			return object.Nil, nil
		}
		sessionObj := args[0]
		sess, ok := sessionObj.Env.(picnet.Session)
		if !ok || sess == nil {
			return nil, &Error{Kind: "IOError", Message: "PicoceciREPL serve: argument must be a TCPSession", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		// Use a bufio.Reader so the runner gets line-buffered input.
		r := bufio.NewReader(sess)
		fn(r, sess)
		return object.Nil, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("PicoceciREPL"), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// LED global
// ---------------------------------------------------------------------------

// makeLEDObject creates the LED singleton picoceci object backed by driver.
func makeLEDObject(driver LEDDriver) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     driver,
	}

	o.Methods["on"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).On()
		return object.Nil, nil
	}}

	o.Methods["off"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).Off()
		return object.Nil, nil
	}}

	o.Methods["toggle"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).Toggle()
		return object.Nil, nil
	}}

	o.Methods["red"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).Red()
		return object.Nil, nil
	}}

	o.Methods["green"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).Green()
		return object.Nil, nil
	}}

	o.Methods["blue"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).Blue()
		return object.Nil, nil
	}}

	o.Methods["white"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).White()
		return object.Nil, nil
	}}

	o.Methods["rgb:green:blue:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) != 3 {
			return nil, &Error{Kind: "LEDError", Message: "LED rgb:green:blue: requires three Integer arguments (0-255)", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		vals := [3]uint8{}
		for i, arg := range args {
			if arg == nil || arg.Kind != object.KindSmallInt || arg.IVal < 0 || arg.IVal > 255 {
				return nil, &Error{Kind: "LEDError", Message: "LED rgb:green:blue: arguments must be Integers in 0-255", Pos: ast.Pos{Line: 1, Col: 1}}
			}
			vals[i] = uint8(arg.IVal)
		}
		self.Env.(LEDDriver).RGB(vals[0], vals[1], vals[2])
		return object.Nil, nil
	}}

	// LED blinkEvery: ms — start periodic blink at ms interval.
	o.Methods["blinkEvery:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) == 0 || args[0] == nil || args[0].Kind != object.KindSmallInt {
			return nil, &Error{Kind: "LEDError", Message: "LED blinkEvery: argument must be an Integer (milliseconds)", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		self.Env.(LEDDriver).BlinkEvery(int(args[0].IVal))
		return object.Nil, nil
	}}

	o.Methods["stopBlink"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		self.Env.(LEDDriver).StopBlink()
		return object.Nil, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("LED"), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// SDCard / File / Directory / Path globals
// ---------------------------------------------------------------------------

const sdcardRootPath = "/sdcard/"

func makeSDCardObject() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}

	o.Methods["mounted"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.BoolObject(sdcard.IsMounted()), nil
	}}

	o.Methods["status"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		if sdcard.IsMounted() {
			return object.SymbolObject("mounted"), nil
		}
		return object.SymbolObject("notMounted"), nil
	}}

	o.Methods["root"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject(sdcardRootPath), nil
	}}

	o.Methods["mount:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		mountPoint, err := requireStringArg(args, 0, "SDCard mount: path must be a String")
		if err != nil {
			return nil, err
		}
		if err := sdcard.Mount(mountPoint); err != nil {
			return nil, ioError("SDCard mount: " + err.Error())
		}
		return object.Nil, nil
	}}

	o.Methods["unmount"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		if err := sdcard.Unmount(); err != nil {
			return nil, ioError("SDCard unmount: " + err.Error())
		}
		return object.Nil, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("SDCard"), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// File global
// ---------------------------------------------------------------------------

// fileObjectData holds the lazy SourceRunner for File runContents:.
type fileObjectData struct {
	runner SourceRunner
}

func makeFileClass(data *fileObjectData) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     data,
	}

	o.Methods["exists:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "File exists: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if !fsys.Exists(path) {
			return object.False, nil
		}
		info, statErr := fsys.Stat(path)
		if statErr != nil {
			return object.False, nil
		}
		return object.BoolObject(!info.IsDir()), nil
	}}

	o.Methods["read:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "File read: path must be a String")
		if err != nil {
			return nil, err
		}
		data, err := sdcard.ReadFile(path)
		if err != nil {
			return nil, ioError("File read: " + err.Error())
		}
		return object.StringObject(string(data)), nil
	}}

	o.Methods["write:to:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		data, err := requireBytesArg(args, 0, "File write:to: first argument must be a String or ByteArray")
		if err != nil {
			return nil, err
		}
		path, err := requireStringArg(args, 1, "File write:to: path must be a String")
		if err != nil {
			return nil, err
		}
		if err := sdcard.WriteFile(path, data); err != nil {
			return nil, ioError("File write:to: " + err.Error())
		}
		return self, nil
	}}

	// write:data: is the natural keyword-message form: path first, data second.
	o.Methods["write:data:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "File write:data: path must be a String")
		if err != nil {
			return nil, err
		}
		data, err := requireBytesArg(args, 1, "File write:data: data must be a String or ByteArray")
		if err != nil {
			return nil, err
		}
		if err := sdcard.WriteFile(path, data); err != nil {
			return nil, ioError("File write:data: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["append:to:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		data, err := requireBytesArg(args, 0, "File append:to: first argument must be a String or ByteArray")
		if err != nil {
			return nil, err
		}
		path, err := requireStringArg(args, 1, "File append:to: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		f, openErr := fsys.Open(path, sdcard.ModeAppend)
		if openErr != nil {
			return nil, ioError("File append:to: " + openErr.Error())
		}
		defer f.Close()
		if _, writeErr := f.Write(data); writeErr != nil {
			return nil, ioError("File append:to: " + writeErr.Error())
		}
		return self, nil
	}}

	o.Methods["delete:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "File delete: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if err := fsys.Remove(path); err != nil {
			return nil, ioError("File delete: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["move:to:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		oldPath, err := requireStringArg(args, 0, "File move:to: source path must be a String")
		if err != nil {
			return nil, err
		}
		newPath, err := requireStringArg(args, 1, "File move:to: destination path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if err := fsys.Rename(oldPath, newPath); err != nil {
			return nil, ioError("File move:to: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["size:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "File size: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		info, err := fsys.Stat(path)
		if err != nil {
			return nil, ioError("File size: " + err.Error())
		}
		return object.IntObject(info.Size()), nil
	}}

	// runContents: reads a file and evaluates its contents in the current environment.
	// All definitions (let, Object, etc.) become available in the global namespace.
	o.Methods["runContents:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		fd, ok := self.Env.(*fileObjectData)
		if !ok || fd == nil || fd.runner == nil {
			return nil, &Error{Kind: "FileError", Message: "File not initialized; call SetFileRunner after creating interpreter", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		path, err := requireStringArg(args, 0, "File runContents: path must be a String")
		if err != nil {
			return nil, err
		}
		// Read the file contents
		data, err := sdcard.ReadFile(path)
		if err != nil {
			return nil, ioError("File runContents: " + err.Error())
		}
		// Parse and evaluate the source in the current global environment
		source := string(data)
		result, evalErr := fd.runner.EvalSource(source)
		if evalErr != nil {
			return nil, &Error{Kind: "FileError", Message: "File runContents: " + evalErr.Error(), Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return result, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("File"), nil
	}}

	return o
}

func makeDirectoryClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}

	o.Methods["entries:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Directory entries: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		entries, err := fsys.ReadDir(path)
		if err != nil {
			return nil, ioError("Directory entries: " + err.Error())
		}
		items := make([]*object.Object, len(entries))
		for i, entry := range entries {
			items[i] = object.StringObject(entry.Name())
		}
		return &object.Object{Kind: object.KindArray, Items: items}, nil
	}}

	o.Methods["exists:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Directory exists: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if !fsys.Exists(path) {
			return object.False, nil
		}
		info, statErr := fsys.Stat(path)
		if statErr != nil {
			return object.False, nil
		}
		return object.BoolObject(info.IsDir()), nil
	}}

	o.Methods["create:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Directory create: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if err := fsys.Mkdir(path); err != nil {
			return nil, ioError("Directory create: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["createAll:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Directory createAll: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if err := fsys.MkdirAll(path); err != nil {
			return nil, ioError("Directory createAll: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["delete:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Directory delete: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if err := fsys.Remove(path); err != nil {
			return nil, ioError("Directory delete: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["deleteAll:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Directory deleteAll: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		if err := fsys.RemoveAll(path); err != nil {
			return nil, ioError("Directory deleteAll: " + err.Error())
		}
		return self, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Directory"), nil
	}}

	return o
}

func makePathClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}

	o.Methods["from:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Path from: argument must be a String")
		if err != nil {
			return nil, err
		}
		return makePathObject(sdcard.PathFrom(path)), nil
	}}

	o.Methods["exists:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		path, err := requireStringArg(args, 0, "Path exists: path must be a String")
		if err != nil {
			return nil, err
		}
		fsys, err := requireSDCardFS()
		if err != nil {
			return nil, err
		}
		return object.BoolObject(fsys.Exists(path)), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Path"), nil
	}}

	return o
}

func makePathObject(path *sdcard.Path) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     path,
	}

	o.Methods["basename"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject(self.Env.(*sdcard.Path).Basename()), nil
	}}

	o.Methods["dirname"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return makePathObject(self.Env.(*sdcard.Path).Dirname()), nil
	}}

	o.Methods["extension"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject(self.Env.(*sdcard.Path).Extension()), nil
	}}

	o.Methods["stem"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject(self.Env.(*sdcard.Path).Stem()), nil
	}}

	o.Methods["asString"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject(self.Env.(*sdcard.Path).String()), nil
	}}

	o.Methods["isAbsolute"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.BoolObject(self.Env.(*sdcard.Path).IsAbsolute()), nil
	}}

	o.Methods["/"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		child, err := requireStringArg(args, 0, "Path / argument must be a String")
		if err != nil {
			return nil, err
		}
		return makePathObject(self.Env.(*sdcard.Path).Join(child)), nil
	}}

	o.Methods[","] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		suffix, err := requireStringArg(args, 0, "Path , argument must be a String")
		if err != nil {
			return nil, err
		}
		return makePathObject(self.Env.(*sdcard.Path).WithSuffix(suffix)), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject(self.Env.(*sdcard.Path).String()), nil
	}}

	return o
}

func requireSDCardFS() (sdcard.FileSystem, error) {
	fsys := sdcard.FS()
	if fsys == nil {
		return nil, ioError("SD card is not mounted")
	}
	return fsys, nil
}

func requireStringArg(args []*object.Object, idx int, message string) (string, error) {
	if idx >= len(args) || args[idx] == nil || args[idx].Kind != object.KindString {
		return "", &Error{Kind: "IOError", Message: message, Pos: ast.Pos{Line: 1, Col: 1}}
	}
	return args[idx].SVal, nil
}

func requireBytesArg(args []*object.Object, idx int, message string) ([]byte, error) {
	if idx >= len(args) || args[idx] == nil {
		return nil, &Error{Kind: "IOError", Message: message, Pos: ast.Pos{Line: 1, Col: 1}}
	}
	switch args[idx].Kind {
	case object.KindString:
		return []byte(args[idx].SVal), nil
	case object.KindByteArray:
		return append([]byte(nil), args[idx].Bytes...), nil
	default:
		return nil, &Error{Kind: "IOError", Message: message, Pos: ast.Pos{Line: 1, Col: 1}}
	}
}

func ioError(message string) *Error {
	return &Error{Kind: "IOError", Message: message, Pos: ast.Pos{Line: 1, Col: 1}}
}

// ---------------------------------------------------------------------------
// Timestamp class and instances
// ---------------------------------------------------------------------------

// timestampData holds milliseconds since boot for a Timestamp instance.
type timestampData struct{ ms int64 }

// makeTimestampClass creates the Timestamp class singleton.
// Timestamp now returns a Timestamp instance capturing the current tick count.
func makeTimestampClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}

	o.Methods["now"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return makeTimestampInstance(int64(freertos.GetTickCount())), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Timestamp"), nil
	}}

	return o
}

// makeTimestampInstance creates a Timestamp picoceci object for ms.
func makeTimestampInstance(ms int64) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     &timestampData{ms: ms},
	}

	o.Methods["asMilliseconds"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.IntObject(self.Env.(*timestampData).ms), nil
	}}

	// ts1 - ts2 → Duration
	o.Methods["-"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td := self.Env.(*timestampData)
		if len(args) == 0 || args[0] == nil {
			return nil, &Error{Kind: "TypeError", Message: "Timestamp - requires a Timestamp argument", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		other, ok := args[0].Env.(*timestampData)
		if !ok {
			return nil, &Error{Kind: "TypeError", Message: "Timestamp - argument must be a Timestamp", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return makeDurationInstance(td.ms - other.ms), nil
	}}

	o.Methods["<"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td := self.Env.(*timestampData)
		if len(args) > 0 && args[0] != nil {
			if other, ok := args[0].Env.(*timestampData); ok {
				return object.BoolObject(td.ms < other.ms), nil
			}
		}
		return object.False, nil
	}}

	o.Methods[">"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td := self.Env.(*timestampData)
		if len(args) > 0 && args[0] != nil {
			if other, ok := args[0].Env.(*timestampData); ok {
				return object.BoolObject(td.ms > other.ms), nil
			}
		}
		return object.False, nil
	}}

	o.Methods["="] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td := self.Env.(*timestampData)
		if len(args) > 0 && args[0] != nil {
			if other, ok := args[0].Env.(*timestampData); ok {
				return object.BoolObject(td.ms == other.ms), nil
			}
		}
		return object.False, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		td := self.Env.(*timestampData)
		return object.StringObject(fmt.Sprintf("Timestamp(%dms)", td.ms)), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// Duration class and instances
// ---------------------------------------------------------------------------

// durationData holds milliseconds for a Duration instance.
type durationData struct{ ms int64 }

// makeDurationClass creates the Duration class singleton.
// Duration ms: n creates a Duration instance for n milliseconds.
func makeDurationClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}

	o.Methods["ms:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) == 0 || args[0] == nil || args[0].Kind != object.KindSmallInt {
			return nil, &Error{Kind: "TypeError", Message: "Duration ms: requires an Integer argument", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return makeDurationInstance(args[0].IVal), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("Duration"), nil
	}}

	return o
}

// makeDurationInstance creates a Duration picoceci object for ms milliseconds.
func makeDurationInstance(ms int64) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     &durationData{ms: ms},
	}

	o.Methods["asMilliseconds"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.IntObject(self.Env.(*durationData).ms), nil
	}}

	o.Methods["asSeconds"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		return object.FloatObject(float64(dd.ms) / 1000.0), nil
	}}

	o.Methods["+"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		if len(args) == 0 || args[0] == nil {
			return nil, &Error{Kind: "TypeError", Message: "Duration + requires a Duration argument", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		other, ok := args[0].Env.(*durationData)
		if !ok {
			return nil, &Error{Kind: "TypeError", Message: "Duration + argument must be a Duration", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return makeDurationInstance(dd.ms + other.ms), nil
	}}

	o.Methods["-"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		if len(args) == 0 || args[0] == nil {
			return nil, &Error{Kind: "TypeError", Message: "Duration - requires a Duration argument", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		other, ok := args[0].Env.(*durationData)
		if !ok {
			return nil, &Error{Kind: "TypeError", Message: "Duration - argument must be a Duration", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		return makeDurationInstance(dd.ms - other.ms), nil
	}}

	o.Methods["<"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		if len(args) > 0 && args[0] != nil {
			if other, ok := args[0].Env.(*durationData); ok {
				return object.BoolObject(dd.ms < other.ms), nil
			}
		}
		return object.False, nil
	}}

	o.Methods[">"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		if len(args) > 0 && args[0] != nil {
			if other, ok := args[0].Env.(*durationData); ok {
				return object.BoolObject(dd.ms > other.ms), nil
			}
		}
		return object.False, nil
	}}

	o.Methods["="] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		if len(args) > 0 && args[0] != nil {
			if other, ok := args[0].Env.(*durationData); ok {
				return object.BoolObject(dd.ms == other.ms), nil
			}
		}
		return object.False, nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
		dd := self.Env.(*durationData)
		if dd.ms < 0 {
			return object.StringObject(fmt.Sprintf("-%dms", -dd.ms)), nil
		}
		if dd.ms >= 1000 {
			return object.StringObject(fmt.Sprintf("%.3fs", float64(dd.ms)/1000.0)), nil
		}
		return object.StringObject(fmt.Sprintf("%dms", dd.ms)), nil
	}}

	return o
}

// ---------------------------------------------------------------------------
// TaskSupervisor global
// ---------------------------------------------------------------------------

// makeTaskSupervisorObject creates the TaskSupervisor singleton.
// It shares taskObjectData with Task; SetTaskCaller wires both simultaneously.
func makeTaskSupervisorObject(data *taskObjectData) *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
		Env:     data,
	}

	// TaskSupervisor supervise: aBlock name: aString
	// Spawns aBlock in a goroutine, restarting on error indefinitely.
	// A clean (nil error) return stops supervision.
	o.Methods["supervise:name:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td, ok := self.Env.(*taskObjectData)
		if !ok || td == nil || td.caller == nil {
			return nil, &Error{Kind: "TaskError", Message: "TaskSupervisor not initialized; call SetTaskCaller after creating interpreter", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		if len(args) < 2 {
			return object.Nil, nil
		}
		blk := args[0]
		if blk == nil || blk.Kind != object.KindBlock {
			return nil, &Error{Kind: "TaskError", Message: "TaskSupervisor supervise:name: first argument must be a Block", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		name := ""
		if args[1] != nil {
			name = args[1].SVal
		}
		caller := td.caller
		go func() {
			for {
				_, err := caller.CallBlock(blk, nil)
				if err == nil {
					return // clean exit — stop supervision
				}
				// error — restart
			}
		}()
		return object.SymbolObject(name), nil
	}}

	// TaskSupervisor supervise: aBlock name: aString maxRestarts: n
	// Spawns aBlock in a goroutine, restarting on error up to n times.
	// A clean (nil error) return stops supervision immediately.
	o.Methods["supervise:name:maxRestarts:"] = &object.MethodDef{Native: func(self *object.Object, args []*object.Object) (*object.Object, error) {
		td, ok := self.Env.(*taskObjectData)
		if !ok || td == nil || td.caller == nil {
			return nil, &Error{Kind: "TaskError", Message: "TaskSupervisor not initialized; call SetTaskCaller after creating interpreter", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		if len(args) < 3 {
			return object.Nil, nil
		}
		blk := args[0]
		if blk == nil || blk.Kind != object.KindBlock {
			return nil, &Error{Kind: "TaskError", Message: "TaskSupervisor supervise:name:maxRestarts: first argument must be a Block", Pos: ast.Pos{Line: 1, Col: 1}}
		}
		name := ""
		if args[1] != nil {
			name = args[1].SVal
		}
		var maxRestarts int64
		if args[2] != nil && args[2].Kind == object.KindSmallInt {
			maxRestarts = args[2].IVal
		}
		caller := td.caller
		go func() {
			var restarts int64
			for {
				_, err := caller.CallBlock(blk, nil)
				if err == nil {
					return // clean exit — stop supervision
				}
				restarts++
				if restarts > maxRestarts {
					return
				}
			}
		}()
		return object.SymbolObject(name), nil
	}}

	o.Methods["printString"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		return object.StringObject("TaskSupervisor"), nil
	}}

	return o
}

// BuiltinDispatch handles message sends to primitive types.
// Returns (result, error, handled). If handled is false, the caller should
// look for a method in the receiver's method table.
// This is exported for use by both the tree-walking interpreter and bytecode VM.
func BuiltinDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch recv.Kind {
	case object.KindSmallInt:
		return intDispatch(caller, recv, sel, args, p)
	case object.KindFloat:
		return floatDispatch(recv, sel, args, p)
	case object.KindBool:
		return boolDispatch(caller, recv, sel, args, p)
	case object.KindString:
		return stringDispatch(caller, recv, sel, args, p)
	case object.KindSymbol:
		return symbolDispatch(recv, sel, args, p)
	case object.KindArray:
		return arrayDispatch(caller, recv, sel, args, p)
	case object.KindBlock:
		return blockDispatch(caller, recv, sel, args, p)
	case object.KindNil:
		return nilDispatch(recv, sel, args, p)
	}
	return nil, nil, false
}

// builtinDispatch is the internal version that takes *Interpreter.
// Kept for backward compatibility within the eval package.
func builtinDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	return BuiltinDispatch(interp, recv, sel, args, p)
}

// --- nil --------------------------------------------------------------------

func nilDispatch(recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch sel {
	case "isNil":
		return object.True, nil, true
	case "notNil":
		return object.False, nil, true
	case "printString":
		return object.StringObject("nil"), nil, true
	case "=":
		if len(args) > 0 && args[0].Kind == object.KindNil {
			return object.True, nil, true
		}
		return object.False, nil, true
	case "~=":
		if len(args) > 0 && args[0].Kind == object.KindNil {
			return object.False, nil, true
		}
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- integers ---------------------------------------------------------------

func intDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	a := recv.IVal
	arg0 := func() (*object.Object, bool) {
		if len(args) == 0 {
			return nil, false
		}
		return args[0], true
	}
	numArg := func() (int64, float64, bool, bool) {
		o, ok := arg0()
		if !ok {
			return 0, 0, false, false
		}
		if o.Kind == object.KindSmallInt {
			return o.IVal, 0, true, false
		}
		if o.Kind == object.KindFloat {
			return 0, o.FVal, false, true
		}
		return 0, 0, false, false
	}

	switch sel {
	case "+":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.IntObject(a + iv), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) + fv), nil, true
		}
	case "-":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.IntObject(a - iv), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) - fv), nil, true
		}
	case "*":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.IntObject(a * iv), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) * fv), nil, true
		}
	case "/":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			if iv == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "division by zero", Pos: p}, true
			}
			if a%iv == 0 {
				return object.IntObject(a / iv), nil, true
			}
			return object.FloatObject(float64(a) / float64(iv)), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) / fv), nil, true
		}
	case "//":
		iv, _, isInt, _ := numArg()
		if isInt {
			if iv == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "division by zero", Pos: p}, true
			}
			return object.IntObject(a / iv), nil, true
		}
	case "\\\\":
		iv, _, isInt, _ := numArg()
		if isInt {
			if iv == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "modulo by zero", Pos: p}, true
			}
			return object.IntObject(a % iv), nil, true
		}
	case "=":
		o, ok := arg0()
		if ok {
			if o.Kind == object.KindSmallInt {
				return object.BoolObject(a == o.IVal), nil, true
			}
			if o.Kind == object.KindFloat {
				return object.BoolObject(float64(a) == o.FVal), nil, true
			}
			return object.False, nil, true
		}
	case "~=":
		o, ok := arg0()
		if ok {
			if o.Kind == object.KindSmallInt {
				return object.BoolObject(a != o.IVal), nil, true
			}
			return object.True, nil, true
		}
	case "<":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a < iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) < fv), nil, true
		}
	case ">":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a > iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) > fv), nil, true
		}
	case "<=":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a <= iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) <= fv), nil, true
		}
	case ">=":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a >= iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) >= fv), nil, true
		}
	case "abs":
		if a < 0 {
			return object.IntObject(-a), nil, true
		}
		return recv, nil, true
	case "negated":
		return object.IntObject(-a), nil, true
	case "sqrt":
		return object.FloatObject(math.Sqrt(float64(a))), nil, true
	case "floor":
		return recv, nil, true
	case "ceiling":
		return recv, nil, true
	case "rounded":
		return recv, nil, true
	case "asFloat":
		return object.FloatObject(float64(a)), nil, true
	case "asInteger":
		return recv, nil, true
	case "printString":
		return object.StringObject(fmt.Sprintf("%d", a)), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	case "timesRepeat:":
		blk, ok := arg0()
		if ok && blk.Kind == object.KindBlock {
			for i := int64(0); i < a; i++ {
				if _, err := caller.CallBlock(blk, nil); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	case "to:do:":
		if len(args) == 2 {
			limit := args[0]
			blk := args[1]
			if limit.Kind == object.KindSmallInt && blk.Kind == object.KindBlock {
				for i := a; i <= limit.IVal; i++ {
					if _, err := caller.CallBlock(blk, []*object.Object{object.IntObject(i)}); err != nil {
						return nil, err, true
					}
				}
				return recv, nil, true
			}
		}
	case "to:":
		// Returns a range-like — for now return self (placeholder)
		return recv, nil, true
	}
	return nil, nil, false
}

// --- floats -----------------------------------------------------------------

func floatDispatch(recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	a := recv.FVal
	numArg := func() (float64, bool) {
		if len(args) == 0 {
			return 0, false
		}
		o := args[0]
		if o.Kind == object.KindFloat {
			return o.FVal, true
		}
		if o.Kind == object.KindSmallInt {
			return float64(o.IVal), true
		}
		return 0, false
	}

	switch sel {
	case "+":
		b, ok := numArg()
		if ok {
			return object.FloatObject(a + b), nil, true
		}
	case "-":
		b, ok := numArg()
		if ok {
			return object.FloatObject(a - b), nil, true
		}
	case "*":
		b, ok := numArg()
		if ok {
			return object.FloatObject(a * b), nil, true
		}
	case "/":
		b, ok := numArg()
		if ok {
			if b == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "division by zero", Pos: p}, true
			}
			return object.FloatObject(a / b), nil, true
		}
	case "=":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a == b), nil, true
		}
		return object.False, nil, true
	case "<":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a < b), nil, true
		}
	case ">":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a > b), nil, true
		}
	case "<=":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a <= b), nil, true
		}
	case ">=":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a >= b), nil, true
		}
	case "abs":
		if a < 0 {
			return object.FloatObject(-a), nil, true
		}
		return recv, nil, true
	case "negated":
		return object.FloatObject(-a), nil, true
	case "sqrt":
		return object.FloatObject(math.Sqrt(a)), nil, true
	case "floor":
		return object.IntObject(int64(math.Floor(a))), nil, true
	case "ceiling":
		return object.IntObject(int64(math.Ceil(a))), nil, true
	case "rounded":
		return object.IntObject(int64(math.Round(a))), nil, true
	case "printString":
		return object.StringObject(fmt.Sprintf("%g", a)), nil, true
	case "asFloat":
		return recv, nil, true
	case "asInteger":
		return object.IntObject(int64(a)), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- booleans ---------------------------------------------------------------

func boolDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	b := recv.BVal
	switch sel {
	case "ifTrue:":
		if b && len(args) > 0 && args[0].Kind == object.KindBlock {
			res, err := caller.CallBlock(args[0], nil)
			return res, err, true
		}
		return object.Nil, nil, true
	case "ifFalse:":
		if !b && len(args) > 0 && args[0].Kind == object.KindBlock {
			res, err := caller.CallBlock(args[0], nil)
			return res, err, true
		}
		return object.Nil, nil, true
	case "ifTrue:ifFalse:":
		if len(args) == 2 {
			blk := args[0]
			if !b {
				blk = args[1]
			}
			if blk.Kind == object.KindBlock {
				res, err := caller.CallBlock(blk, nil)
				return res, err, true
			}
		}
		return object.Nil, nil, true
	case "ifFalse:ifTrue:":
		if len(args) == 2 {
			blk := args[1]
			if !b {
				blk = args[0]
			}
			if blk.Kind == object.KindBlock {
				res, err := caller.CallBlock(blk, nil)
				return res, err, true
			}
		}
		return object.Nil, nil, true
	case "not":
		return object.BoolObject(!b), nil, true
	case "&":
		if len(args) > 0 {
			return object.BoolObject(b && args[0].Truthy()), nil, true
		}
	case "|":
		if len(args) > 0 {
			return object.BoolObject(b || args[0].Truthy()), nil, true
		}
	case "=":
		if len(args) > 0 && args[0].Kind == object.KindBool {
			return object.BoolObject(b == args[0].BVal), nil, true
		}
		return object.False, nil, true
	case "printString":
		if b {
			return object.StringObject("true"), nil, true
		}
		return object.StringObject("false"), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- strings ----------------------------------------------------------------

func stringDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	s := recv.SVal
	switch sel {
	case "size":
		return object.IntObject(int64(len([]rune(s)))), nil, true
	case "printString":
		return object.StringObject("'" + strings.ReplaceAll(s, "'", "''") + "'"), nil, true
	case "displayString":
		return object.StringObject(s), nil, true
	case ",":
		if len(args) > 0 {
			other := args[0].SVal
			return object.StringObject(s + other), nil, true
		}
	case "reversed":
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return object.StringObject(string(runes)), nil, true
	case "asUppercase":
		return object.StringObject(strings.ToUpper(s)), nil, true
	case "asLowercase":
		return object.StringObject(strings.ToLower(s)), nil, true
	case "asSymbol":
		return object.SymbolObject(s), nil, true
	case "asBytes":
		return object.ByteArrayObject([]byte(s)), nil, true
	case "=":
		if len(args) > 0 {
			if args[0].Kind == object.KindString {
				return object.BoolObject(s == args[0].SVal), nil, true
			}
			return object.False, nil, true
		}
	case "~=":
		if len(args) > 0 {
			if args[0].Kind == object.KindString {
				return object.BoolObject(s != args[0].SVal), nil, true
			}
			return object.True, nil, true
		}
	case "<":
		if len(args) > 0 && args[0].Kind == object.KindString {
			return object.BoolObject(s < args[0].SVal), nil, true
		}
	case ">":
		if len(args) > 0 && args[0].Kind == object.KindString {
			return object.BoolObject(s > args[0].SVal), nil, true
		}
	case "trimSeparators":
		return object.StringObject(strings.TrimSpace(s)), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	case "at:":
		if len(args) > 0 && args[0].Kind == object.KindSmallInt {
			runes := []rune(s)
			idx64 := args[0].IVal
			nRunes := int64(len(runes))
			if idx64 < 1 || idx64 > nRunes || idx64 > math.MaxInt {
				return nil, &Error{Kind: "IndexOutOfBounds", Message: fmt.Sprintf("index %d out of bounds (size %d)", idx64, nRunes), Pos: p}, true
			}
			return object.CharObject(runes[int(idx64)-1]), nil, true
		}
	case "copyFrom:to:":
		if len(args) == 2 && args[0].Kind == object.KindSmallInt && args[1].Kind == object.KindSmallInt {
			runes := []rune(s)
			n := int64(len(runes))
			start64 := args[0].IVal - 1
			stop64 := args[1].IVal
			if start64 < 0 {
				start64 = 0
			}
			if stop64 > n {
				stop64 = n
			}
			if start64 > stop64 || stop64 > math.MaxInt {
				return object.StringObject(""), nil, true
			}
			return object.StringObject(string(runes[int(start64):int(stop64)])), nil, true
		}
	case "includesSubString:":
		if len(args) > 0 && args[0].Kind == object.KindString {
			return object.BoolObject(strings.Contains(s, args[0].SVal)), nil, true
		}
	case "asInteger":
		if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			return object.IntObject(v), nil, true
		}
		return object.Nil, nil, true
	case "asFloat":
		if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return object.FloatObject(v), nil, true
		}
		return object.Nil, nil, true
	case "do:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for _, r := range s {
				if _, err := caller.CallBlock(args[0], []*object.Object{object.CharObject(r)}); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	}
	return nil, nil, false
}

// --- symbols ----------------------------------------------------------------

func symbolDispatch(recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch sel {
	case "asString":
		return object.StringObject(recv.SVal), nil, true
	case "printString":
		return object.StringObject("#" + recv.SVal), nil, true
	case "=":
		if len(args) > 0 && args[0].Kind == object.KindSymbol {
			return object.BoolObject(recv.SVal == args[0].SVal), nil, true
		}
		return object.False, nil, true
	case "asSymbol":
		return recv, nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- arrays -----------------------------------------------------------------

func arrayDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	items := recv.Items
	switch sel {
	case "size":
		return object.IntObject(int64(len(items))), nil, true
	case "at:":
		if len(args) > 0 && args[0].Kind == object.KindSmallInt {
			idx64 := args[0].IVal
			nItems := int64(len(items))
			if idx64 < 1 || idx64 > nItems || idx64 > math.MaxInt {
				return nil, &Error{Kind: "IndexOutOfBounds", Message: fmt.Sprintf("index %d out of bounds (size %d)", idx64, nItems), Pos: p}, true
			}
			return items[int(idx64)-1], nil, true
		}
	case "at:put:":
		if len(args) == 2 && args[0].Kind == object.KindSmallInt {
			idx64 := args[0].IVal
			nItems := int64(len(items))
			if idx64 < 1 || idx64 > nItems || idx64 > math.MaxInt {
				return nil, &Error{Kind: "IndexOutOfBounds", Message: fmt.Sprintf("index %d out of bounds", idx64), Pos: p}, true
			}
			items[int(idx64)-1] = args[1]
			return args[1], nil, true
		}
	case "first":
		if len(items) > 0 {
			return items[0], nil, true
		}
		return object.Nil, nil, true
	case "last":
		if len(items) > 0 {
			return items[len(items)-1], nil, true
		}
		return object.Nil, nil, true
	case "do:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for _, item := range items {
				if _, err := caller.CallBlock(args[0], []*object.Object{item}); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	case "collect:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			result := object.ArrayObject(len(items))
			for i, item := range items {
				v, err := caller.CallBlock(args[0], []*object.Object{item})
				if err != nil {
					return nil, err, true
				}
				result.Items[i] = v
			}
			return result, nil, true
		}
	case "select:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			var result []*object.Object
			for _, item := range items {
				v, err := caller.CallBlock(args[0], []*object.Object{item})
				if err != nil {
					return nil, err, true
				}
				if v.Truthy() {
					result = append(result, item)
				}
			}
			arr := &object.Object{Kind: object.KindArray, Items: result}
			return arr, nil, true
		}
	case "inject:into:":
		if len(args) == 2 && args[1].Kind == object.KindBlock {
			acc := args[0]
			for _, item := range items {
				v, err := caller.CallBlock(args[1], []*object.Object{acc, item})
				if err != nil {
					return nil, err, true
				}
				acc = v
			}
			return acc, nil, true
		}
	case "detect:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for _, item := range items {
				v, err := caller.CallBlock(args[0], []*object.Object{item})
				if err != nil {
					return nil, err, true
				}
				if v.Truthy() {
					return item, nil, true
				}
			}
			return nil, &Error{Kind: "ElementNotFound", Message: "detect: no element satisfies the block", Pos: p}, true
		}
	case "withIndexDo:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for i, item := range items {
				if _, err := caller.CallBlock(args[0], []*object.Object{object.IntObject(int64(i + 1)), item}); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	case "printString":
		var parts []string
		for _, item := range items {
			parts = append(parts, item.PrintString())
		}
		return object.StringObject("(" + strings.Join(parts, " ") + " )"), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- blocks -----------------------------------------------------------------

func blockDispatch(caller BlockCaller, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch sel {
	case "value":
		res, err := caller.CallBlock(recv, nil)
		return res, err, true
	case "value:":
		res, err := caller.CallBlock(recv, args)
		return res, err, true
	case "value:value:":
		res, err := caller.CallBlock(recv, args)
		return res, err, true
	case "valueWithArguments:":
		if len(args) > 0 && args[0].Kind == object.KindArray {
			res, err := caller.CallBlock(recv, args[0].Items)
			return res, err, true
		}
	case "whileTrue:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for {
				cond, err := caller.CallBlock(recv, nil)
				if err != nil {
					return nil, err, true
				}
				if !cond.Truthy() {
					break
				}
				if _, err = caller.CallBlock(args[0], nil); err != nil {
					return nil, err, true
				}
			}
			return object.Nil, nil, true
		}
	case "whileTrue":
		for {
			cond, err := caller.CallBlock(recv, nil)
			if err != nil {
				return nil, err, true
			}
			if !cond.Truthy() {
				break
			}
		}
		return object.Nil, nil, true
	case "whileFalse:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for {
				cond, err := caller.CallBlock(recv, nil)
				if err != nil {
					return nil, err, true
				}
				if cond.Truthy() {
					break
				}
				if _, err = caller.CallBlock(args[0], nil); err != nil {
					return nil, err, true
				}
			}
			return object.Nil, nil, true
		}
	case "on:do:":
		if len(args) == 2 {
			_, err := caller.CallBlock(recv, nil)
			if err != nil {
				// Check if it's a picoceci Error — if so, call the handler block.
				if _, ok := err.(*Error); ok {
					errObj := object.StringObject(err.Error())
					res, herr := caller.CallBlock(args[1], []*object.Object{errObj})
					return res, herr, true
				}
				return nil, err, true
			}
			return object.Nil, nil, true
		}
	case "ensure:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			res, err := caller.CallBlock(recv, nil)
			_, _ = caller.CallBlock(args[0], nil) // always run
			return res, err, true
		}
	case "printString":
		return object.StringObject("a BlockClosure"), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}
