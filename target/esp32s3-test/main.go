//go:build tinygo

// Minimal test for ESP32-S3 serial output
package main

import (
	"time"
)

func main() {
	// Wait for USB to initialize
	time.Sleep(2 * time.Second)

	// Simple loop printing to serial
	for {
		println("hello from esp32s3")
		time.Sleep(time.Second)
	}
}
