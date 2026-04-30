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

// NOTE: This file is a skeleton.  The actual TinyGo machine/sdcard imports
// and FreeRTOS bindings will be added in Phase 5 of the implementation.
//
// The build tag ensures this file is ONLY compiled by TinyGo, keeping the
// desktop build (go build ./...) clean.

func main() {
	// Phase 5: Initialise UART0, mount SD card, start REPL.
	// Placeholder — will be filled in during Phase 5 implementation.
	for {
		// idle loop so TinyGo doesn't complain about empty main
	}
}
