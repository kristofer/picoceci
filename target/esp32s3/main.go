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
	"strings"
	"time"

	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/module"
	"github.com/kristofer/picoceci/pkg/parser"
	"github.com/kristofer/picoceci/pkg/sdcard"
	"github.com/kristofer/picoceci/pkg/tinygo"
)

const version = "0.1.0-dev"

// transcriptPlaceholder is a temporary Transcript sink.
// Replace this writer with a Canal TCP session writer when ready.
type transcriptPlaceholder struct{}

func (w *transcriptPlaceholder) Write(p []byte) (int, error) {
	return len(p), nil
}

func main() {
	// Wait for USB CDC to initialize
	time.Sleep(2 * time.Second)

	// Initialize console
	console := tinygo.NewConsole()
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

	write(console, "Ready.\n\n")

	// Start REPL
	runREPL(console, loader)
}

// runREPL runs the interactive REPL.
// Uses fresh VM per expression to minimize memory accumulation.
//
// Paste mode: type "---" alone on a line to enter paste mode; all
// subsequent lines are buffered.  Type "---" again to execute the
// buffered program as a single unit.  This lets you paste multi-line
// programs over the USB serial interface without triggering a parse
// error on every incomplete line.
func runREPL(console tinygo.Console, loader *module.Loader) {
	var buf strings.Builder
	inPaste := false

	for {
		if inPaste {
			write(console, "... ")
		} else {
			write(console, "> ")
		}

		line, err := console.ReadLine()
		if err != nil {
			write(console, "\nGoodbye!\n")
			break
		}

		// "---" toggles paste mode.
		if line == "---" {
			if !inPaste {
				inPaste = true
				buf.Reset()
				write(console, "(paste mode on: type '---' to run)\n")
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

// execSource parses, compiles, and runs src, writing results to console.
func execSource(console tinygo.Console, src string, loader *module.Loader) {
	// Parse
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		write(console, "parse: "+err.Error()+"\n")
		return
	}

	// Compile
	c := bytecode.NewCompilerWithLoader(loader)
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		write(console, "compile: "+err.Error()+"\n")
		return
	}

	// Run with fresh VM each time.
	// Console stays on USB serial; Transcript is currently a placeholder sink.
	vm := bytecode.NewVMWithSinks(eval.GlobalSinks{
		ConsoleWriter:    console,
		TranscriptWriter: &transcriptPlaceholder{},
	})
	vm.SetBlocks(c.GetBlocks())
	vm.AddGlobals(c.GetGlobals())
	result, err := vm.Run(chunk)
	if err != nil {
		write(console, "error: "+err.Error()+"\n")
		return
	}

	// Print result
	if result != nil {
		write(console, "=> "+result.PrintString()+"\n")
	}
}

// write is a helper to write a string to the console.
func write(c tinygo.Console, s string) {
	c.Write([]byte(s))
}
