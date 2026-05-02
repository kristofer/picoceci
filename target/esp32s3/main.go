//go:build tinygo

// Command esp32s3 is the TinyGo entry point for picoceci on the ESP32-S3.
//
// It:
//  1. Initialises UART0 at 115200 baud for console I/O.
//  2. Mounts the SD card at /sdcard/ (if available).
//  3. Starts the picoceci REPL on UART0.
//
// Build with:
//
//	tinygo build -target=esp32s3-generic ./target/esp32s3
//
// Flash with:
//
//	tinygo flash -target=esp32s3-generic -port=/dev/cu.usbmodem11201 ./target/esp32s3
//
// See IMPLEMENTATION_PLAN.md Phase 5 for implementation notes.
package main

import (
	"time"

	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/module"
	"github.com/kristofer/picoceci/pkg/parser"
	"github.com/kristofer/picoceci/pkg/sdcard"
	"github.com/kristofer/picoceci/pkg/tinygo"
)

const version = "0.1.0-dev"

func main() {
	// Wait for USB CDC to initialize before printing
	time.Sleep(2 * time.Second)

	// Debug: use println directly to test basic serial output
	println("=== picoceci booting ===")

	// Initialize console
	console := tinygo.NewConsole()
	println("console initialized")
	write(console, "picoceci starting...\n")
	time.Sleep(1 * time.Second)

	// Try to mount SD card
	if err := sdcard.Mount("/sdcard/"); err != nil {
		write(console, "Warning: SD card not mounted ("+err.Error()+")\n")
		write(console, "Continuing without SD card support.\n")
	} else {
		write(console, "SD card mounted at /sdcard/\n")
	}

	// Set up module resolver
	var resolver *module.Resolver
	if sdcard.IsMounted() {
		resolver = module.NewResolver(sdcard.ReadFile)
	} else {
		// Fallback: resolver with no file access
		resolver = module.NewResolver(func(path string) ([]byte, error) {
			return nil, sdcard.ErrNotMounted
		})
	}
	module.RegisterBuiltins(resolver)
	loader := module.NewLoader(resolver)

	// Create VM
	vm := bytecode.NewVM()

	// Run quick self-test
	write(console, "Running self-test...\n")
	if err := selfTest(); err != nil {
		write(console, "Self-test FAILED: "+err.Error()+"\n")
	} else {
		write(console, "Self-test passed!\n")
	}

	// Start REPL
	runREPL(console, vm, loader)
}

// selfTest runs a quick bytecode VM test.
func selfTest() error {
	src := `| factorial | factorial := [ :x | x <= 1 ifTrue: [ 1 ] ifFalse: [ x * (factorial value: x - 1) ] ]. factorial value: 5.`

	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		return err
	}

	c := bytecode.NewCompiler()
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		return err
	}

	vm := bytecode.NewVM()
	vm.SetBlocks(c.GetBlocks())
	_, err = vm.Run(chunk)
	return err
}

// runREPL runs the interactive REPL.
func runREPL(console tinygo.Console, vm *bytecode.VM, loader *module.Loader) {
	write(console, "\npicoceci "+version+" (TinyGo/ESP32-S3)\n")
	write(console, "Type picoceci expressions. Use Ctrl-C to exit.\n\n")

	for {
		write(console, "picoceci> ")
		line, err := console.ReadLine()
		if err != nil {
			write(console, "\nGoodbye!\n")
			break
		}
		if line == "" {
			continue
		}

		// Parse
		l := lexer.NewString(line)
		p := parser.New(l)
		prog, err := p.ParseProgram()
		if err != nil {
			write(console, "parse error: "+err.Error()+"\n")
			continue
		}

		// Compile
		c := bytecode.NewCompilerWithLoader(loader)
		chunk, err := c.Compile(prog.Statements)
		if err != nil {
			write(console, "compile error: "+err.Error()+"\n")
			continue
		}

		// Run
		vm.SetBlocks(c.GetBlocks())
		vm.AddGlobals(c.GetGlobals())
		result, err := vm.Run(chunk)
		if err != nil {
			write(console, "error: "+err.Error()+"\n")
			continue
		}

		// Print result
		if result != nil {
			write(console, "=> "+result.PrintString()+"\n")
		}
	}
}

// write is a helper to write a string to the console.
func write(c tinygo.Console, s string) {
	c.Write([]byte(s))
}
