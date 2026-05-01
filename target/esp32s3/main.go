//go:build tinygo

// Command esp32s3 is the TinyGo entry point for picoceci on the ESP32-S3.
//
// It:
//  1. Initialises UART0 at 115200 baud for console I/O.
//  2. Mounts the SD card at /sdcard/.
//  3. Starts the picoceci REPL on UART0.
//
// Build with:
//
//	tinygo build -target=esp32-coreboard-v2 ./target/esp32s3
//
// See IMPLEMENTATION_PLAN.md Phase 5 for implementation notes.
package main

import (
	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/parser"
)

// runProgram compiles and runs a picoceci program using the bytecode VM.
func runProgram(src string) error {
	// Parse the source
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		return err
	}

	// Compile to bytecode
	c := bytecode.NewCompiler()
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		return err
	}

	// Create and run VM
	vm := bytecode.NewVM()
	vm.SetBlocks(c.GetBlocks())
	_, err = vm.Run(chunk)
	return err
}

func main() {
	// Phase 3: Simple test using bytecode VM
	// This demonstrates the VM is functional on the target platform.
	//
	// Phase 5 will add: UART0 init, SD card mount, full REPL.

	// Test program: compute 10 factorial
	_ = runProgram(`
		| factorial n result |
		factorial := [ :x |
			x <= 1 ifTrue: [ 1 ] ifFalse: [ x * (factorial value: x - 1) ]
		].
		result := factorial value: 10.
		result.
	`)

	// Keep running (TinyGo requirement)
	for {
		// idle loop
	}
}
