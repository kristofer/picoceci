// Package tinygo provides TinyGo-specific runtime shims for picoceci.
//
// When building for a desktop host (go:build !tinygo), this package
// provides stub implementations that map to standard Go libraries so
// that the full test suite can run without hardware.
//
// When building with TinyGo (go:build tinygo), this package maps to
// TinyGo's machine package and FreeRTOS bindings.
//
// # Console
//
// The Console interface provides platform-independent console I/O:
//
//	c := tinygo.NewConsole()
//	c.Write([]byte("Hello!\n"))
//	line, _ := c.ReadLine()
//
// On TinyGo/ESP32-S3, Console wraps machine.UART0 at 115200 baud.
// On desktop, Console wraps stdin/stdout for testing.
package tinygo
