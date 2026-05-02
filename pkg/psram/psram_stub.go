//go:build !tinygo || !esp32s3_psram

// Package psram provides PSRAM initialization for ESP32-S3 boards with PSRAM.
// This is the stub implementation for non-TinyGo or non-PSRAM builds.
package psram

// Init is a no-op on systems without PSRAM.
func Init() error {
	return nil
}

// Test is a no-op on systems without PSRAM.
func Test() error {
	return nil
}
