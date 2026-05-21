//go:build tinygo

// Command esp32s3 is the TinyGo entry point for picoceci on the ESP32-S3.
//
// Build with:
//
//	tinygo build -target=esp32s3-generic ./target/esp32s3
//
// Flash with:
//
//	tinygo flash -target=esp32s3-generic -port=/dev/cu.usbmodem* ./target/esp32s3
//
// Memory-optimized for ESP32-S3 (~320KB internal SRAM).
// For best results, build with: tinygo flash -gc=leaking ...
package main

import (
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/module"
	picnet "github.com/kristofer/picoceci/pkg/net"
	"github.com/kristofer/picoceci/pkg/object"
	"github.com/kristofer/picoceci/pkg/parser"
	"github.com/kristofer/picoceci/pkg/sdcard"
	"github.com/kristofer/picoceci/pkg/tinygo"
)

const version = "0.3.0-dev"

// wifiSSID and wifiPass are build-time WiFi credentials.
// Override at build time with:
//
//	-ldflags="-X main.wifiSSID=myssid -X main.wifiPass=mypassword"
var (
	wifiSSID = "picoceci-net"
	wifiPass = ""
	wifiPort = 2323
)

const (
	startupSerialReadyTimeout = 2 * time.Second
	startupSerialPollInterval = 20 * time.Millisecond
	tcpAcceptRetryDelay       = 300 * time.Millisecond
)

func main() {
	// Initialize console
	console := tinygo.NewConsole()
	write(console, "boot: console initialized\n")
	write(console, "boot: network mode "+runtimeNetworkMode()+"\n")

	// Initialize board LED
	led := tinygo.NewLED()
	led.On() // blink once to confirm LED is wired
	time.Sleep(100 * time.Millisecond)
	led.Off()

	// Wait for USB CDC host traffic (if any), but never block boot forever.
	if waitForSerialReady(console, startupSerialReadyTimeout) {
		write(console, "boot: usb host activity detected\n")
	} else {
		write(console, "boot: usb wait timeout, continuing\n")
	}
	write(console, "picoceci "+version+" (ESP32-S3)\n")

	// Try to mount SD card (will fail without hardware driver)
	if err := sdcard.Mount("/sdcard/"); err != nil {
		write(console, "SD card: not available\n")
	} else {
		write(console, "SD card: mounted\n")
	}

	// Set up module resolver
	var resolver *module.Resolver
	if sdcard.IsMounted() {
		resolver = module.NewResolver(sdcard.ReadFile)
	} else {
		resolver = module.NewResolver(func(path string) ([]byte, error) {
			return nil, sdcard.ErrNotMounted
		})
	}
	module.RegisterBuiltins(resolver)
	loader := module.NewLoader(resolver)

	// Connect to WiFi.  Continue even if this fails so the serial REPL
	// remains available as a recovery path.
	wifiMgr := picnet.NewManager()
	if wifiSSID != "" {
		write(console, "WiFi connecting (ssid="+wifiSSID+")...\n")
		if err := wifiMgr.Connect(wifiSSID, wifiPass); err != nil {
			write(console, "WiFi error: "+err.Error()+"\n")
			write(console, "WiFi fallback: serial REPL remains available\n")
		} else {
			write(console, "WiFi connected: "+wifiMgr.IPAddress()+"\n")
		}
	} else {
		write(console, "WiFi disabled (empty SSID)\n")
	}

	// Start WiFi TCP listener if connected.
	var tcpListener picnet.Listener
	if wifiMgr.Status() == picnet.WifiStateConnected {
		ln, err := wifiMgr.Listen(wifiPort)
		if err != nil {
			write(console, "TCP listen error: "+err.Error()+"\n")
			write(console, "TCP fallback: serial REPL remains available\n")
		} else {
			tcpListener = ln
			write(console, "TCP REPL on :"+itoa(wifiPort)+"\n")
			go acceptLoop(console, tcpListener, loader, wifiMgr, led)
		}
	} else {
		write(console, "TCP REPL disabled: WiFi state="+wifiMgr.Status().String()+"\n")
	}
	_ = tcpListener

	write(console, "Ready.\n\n")

	// Start serial REPL using the console's line reader so typed input echoes.
	runSerialREPL(console, loader, wifiMgr, led)
}

// waitForSerialReady waits until input appears on USB serial or timeout elapses.
// This gives hosts a chance to attach without forcing a fixed startup delay.
func waitForSerialReady(console tinygo.Console, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if console.Available() > 0 {
			return true
		}
		time.Sleep(startupSerialPollInterval)
	}
	return false
}

// acceptLoop accepts one TCP connection at a time and runs a REPL on it.
// On disconnect it loops back and waits for the next connection.
// The serial Console always remains available for local recovery.
func acceptLoop(console tinygo.Console, ln picnet.Listener, loader *module.Loader, wifiMgr *picnet.Manager, led tinygo.LED) {
	for {
		sess, err := ln.Accept()
		if err != nil {
			errMsg := err.Error()
			write(console, "TCP accept error: "+errMsg+"\n")
			if isClosedListenerError(errMsg) {
				write(console, "TCP listener closed; stopping accept loop\n")
				return
			}
			write(console, "TCP accept retrying...\n")
			time.Sleep(tcpAcceptRetryDelay)
			continue
		}
		write(console, "TCP session from "+sess.RemoteAddr()+"\n")
		runSessionREPL(bufio.NewReader(sess), sess, loader, wifiMgr, led)
		if err := sess.Close(); err != nil {
			write(console, "TCP session close warning: "+err.Error()+"\n")
		}
		write(console, "TCP session ended\n")
	}
}

func isClosedListenerError(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "closed") || strings.Contains(m, "bad file descriptor")
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	if v < 0 {
		return "-" + itoa(-v)
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(b[i:])
}

// runSerialREPL runs the interactive REPL over the local USB console.
// It uses the console's built-in line reader so typed characters echo and
// backspace/enter behave naturally on the serial terminal.
func runSerialREPL(console tinygo.Console, loader *module.Loader, wifiMgr *picnet.Manager, led tinygo.LED) {
	state := newVMState(console, loader, wifiMgr, led)
	var buf strings.Builder
	inPaste := false

	for {
		if inPaste {
			writeStr(console, "... ")
		} else {
			writeStr(console, "> ")
		}

		line, err := console.ReadLine()
		if err != nil {
			writeStr(console, "\nGoodbye!\n")
			break
		}

		// "---" toggles paste mode.
		if line == "---" {
			if !inPaste {
				inPaste = true
				buf.Reset()
				writeStr(console, "(paste mode on: type '---' to run)\n")
			} else {
				inPaste = false
				src := buf.String()
				buf.Reset()
				if src != "" {
					execSource(console, src, state)
				}
			}
			continue
		}

		if inPaste {
			buf.WriteString(line)
			buf.WriteByte('\n')
			continue
		}

		if line == "" {
			continue
		}

		if handleMetaCommand(console, line, state) {
			continue
		}

		execSource(console, line, state)
	}
}

// runSessionREPL runs the interactive REPL for TCP sessions.
// r is the input source; w receives output (Transcript + prompts + results).
// Globals and compiled blocks persist across evaluations within the session
// to retain state (variable assignments, custom methods, etc.).
//
// Paste mode: type "---" alone on a line to enter paste mode; all
// subsequent lines are buffered.  Type "---" again to execute the
// buffered program as a single unit.  This lets you paste multi-line
// programs over the USB serial interface without triggering a parse
// error on every incomplete line.
func runSessionREPL(r io.Reader, w io.Writer, loader *module.Loader, wifiMgr *picnet.Manager, led tinygo.LED) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	state := newVMState(w, loader, wifiMgr, led)
	var buf strings.Builder
	inPaste := false

	for {
		if inPaste {
			writeStr(w, "... ")
		} else {
			writeStr(w, "> ")
		}

		line, err := br.ReadString('\n')
		if err != nil {
			writeStr(w, "\nGoodbye!\n")
			break
		}
		line = strings.TrimRight(line, "\r\n")

		// "---" toggles paste mode.
		if line == "---" {
			if !inPaste {
				inPaste = true
				buf.Reset()
				writeStr(w, "(paste mode on: type '---' to run)\n")
			} else {
				inPaste = false
				src := buf.String()
				buf.Reset()
				if src != "" {
					execSource(w, src, state)
				}
			}
			continue
		}

		if inPaste {
			buf.WriteString(line)
			buf.WriteByte('\n')
			continue
		}

		if line == "" {
			continue
		}

		if handleMetaCommand(w, line, state) {
			continue
		}

		execSource(w, line, state)
	}
}

// vmState holds persistent REPL state (globals and compiled blocks) across
// multiple expression evaluations within a single REPL session.
// This mirrors the desktop cmd/picoceci/main.go pattern and ensures that
// globals like LED, Wifi, Task, etc. are initialised exactly once per session.
type vmState struct {
	globals map[string]*object.Object
	blocks  []*bytecode.CompiledBlock
	loader  *module.Loader
}

// newVMState initialises a vmState with fully-wired sinks.  It calls
// NewVMWithSinks exactly once so that picnet.NewManager() and all other
// singleton allocations occur a single time rather than on every evaluation.
func newVMState(w io.Writer, loader *module.Loader, wifiMgr *picnet.Manager, led tinygo.LED) *vmState {
	vm := bytecode.NewVMWithSinks(eval.GlobalSinks{
		ConsoleWriter:    w,
		TranscriptWriter: w,
		WifiManager:      wifiMgr,
		LEDDriver:        led,
	})
	return &vmState{
		globals: vm.Globals(),
		blocks:  make([]*bytecode.CompiledBlock, 0),
		loader:  loader,
	}
}

// execSource parses, compiles, and runs src, writing results to w.
// It uses state.globals so all session singletons (LED, Wifi, Task, …) are
// available, and persists any new globals/blocks back into state so the next
// call sees them.
func execSource(w io.Writer, src string, state *vmState) {
	// Parse
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		writeStr(w, "parse: "+err.Error()+"\n")
		return
	}

	// Compile with accumulated blocks; top-level vars become globals.
	c := bytecode.NewCompilerWithLoader(state.loader)
	c.SetBlocks(state.blocks)
	c.SetTopLevelVarsAreGlobals(true)
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		writeStr(w, "compile: "+err.Error()+"\n")
		return
	}

	// Run using persistent globals – avoids redundant InitialGlobalsWithSinks.
	vm := bytecode.NewVMWithGlobals(state.globals)
	vm.SetBlocks(c.GetBlocks())
	vm.AddGlobals(c.GetGlobals())
	result, err := vm.Run(chunk)
	if err != nil {
		writeStr(w, "error: "+err.Error()+"\n")
		return
	}

	// Persist updated globals and blocks for next call.
	for name, val := range vm.Globals() {
		state.globals[name] = val
	}
	state.blocks = c.GetBlocks()

	// Print result
	if result != nil {
		writeStr(w, "=> "+result.PrintString()+"\n")
	}
}

// write is a helper to write a string to a tinygo.Console.
func write(c tinygo.Console, s string) {
	c.Write([]byte(s))
}

// writeStr is a helper to write a string to any io.Writer.
func writeStr(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

// handleMetaCommand processes REPL meta-commands that begin with ".".
// Returns true if the line was a meta-command and has been handled.
//
//	.globals   – list all global names currently in the VM state
//	.help      – show available meta-commands
func handleMetaCommand(w io.Writer, line string, state *vmState) bool {
	switch line {
	case ".globals":
		writeStr(w, "globals ("+itoa(len(state.globals))+"):\n")
		for name := range state.globals {
			writeStr(w, "  "+name+"\n")
		}
		return true
	case ".help":
		writeStr(w, "meta-commands: .globals  .help  .version\n")
		return true
	case ".version":
		writeStr(w, "picoceci "+version+"\n")
		return true
	}
	return false
}
