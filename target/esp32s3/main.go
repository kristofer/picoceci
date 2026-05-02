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
func runREPL(console tinygo.Console, loader *module.Loader) {
	for {
		write(console, "> ")
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
			write(console, "parse: "+err.Error()+"\n")
			continue
		}

		// Compile
		c := bytecode.NewCompilerWithLoader(loader)
		chunk, err := c.Compile(prog.Statements)
		if err != nil {
			write(console, "compile: "+err.Error()+"\n")
			continue
		}

		// Run with fresh VM each time
		vm := bytecode.NewVM()
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
