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
			go acceptLoop(console, tcpListener, loader)
		}
	} else {
		write(console, "TCP REPL disabled: WiFi state="+wifiMgr.Status().String()+"\n")
	}
	_ = tcpListener

	write(console, "Ready.\n\n")

	// Start serial REPL using the console's line reader so typed input echoes.
	runSerialREPL(console, loader)
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
func acceptLoop(console tinygo.Console, ln picnet.Listener, loader *module.Loader) {
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
		runSessionREPL(bufio.NewReader(sess), sess, loader)
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
func runSerialREPL(console tinygo.Console, loader *module.Loader) {
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
					execSource(console, src, loader)
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

		execSource(console, line, loader)
	}
}

// runSessionREPL runs the interactive REPL for TCP sessions.
// r is the input source; w receives output (Transcript + prompts + results).
// Uses fresh VM per expression to minimize memory accumulation.
//
// Paste mode: type "---" alone on a line to enter paste mode; all
// subsequent lines are buffered.  Type "---" again to execute the
// buffered program as a single unit.  This lets you paste multi-line
// programs over the USB serial interface without triggering a parse
// error on every incomplete line.
func runSessionREPL(r io.Reader, w io.Writer, loader *module.Loader) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

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
					execSource(w, src, loader)
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

		execSource(w, line, loader)
	}
}

// execSource parses, compiles, and runs src, writing results to w.
// Both Console and Transcript are wired to w so remote sessions see all output.
func execSource(w io.Writer, src string, loader *module.Loader) {
	// Parse
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		writeStr(w, "parse: "+err.Error()+"\n")
		return
	}

	// Compile
	c := bytecode.NewCompilerWithLoader(loader)
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		writeStr(w, "compile: "+err.Error()+"\n")
		return
	}

	// Run with fresh VM each time.
	// Transcript is bound to w (the active session or serial console).
	vm := bytecode.NewVMWithSinks(eval.GlobalSinks{
		ConsoleWriter:    w,
		TranscriptWriter: w,
	})
	vm.SetBlocks(c.GetBlocks())
	vm.AddGlobals(c.GetGlobals())
	result, err := vm.Run(chunk)
	if err != nil {
		writeStr(w, "error: "+err.Error()+"\n")
		return
	}

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
