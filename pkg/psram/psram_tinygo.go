//go:build tinygo && esp32s3_psram

// Package psram provides PSRAM initialization for ESP32-S3 boards with PSRAM.
package psram

// These functions are implemented in C (targets/psram_init.c)
// They must be compiled and linked with the TinyGo binary.

//export psram_init
func psramInitC() int32

//export psram_test
func psramTestC() int32

// Init initializes the PSRAM hardware and configures the MMU to map
// PSRAM at 0x3D000000. This must be called before any heap allocations.
//
// Returns nil on success, or an error if PSRAM initialization fails.
func Init() error {
	if psramInitC() != 0 {
		return ErrInitFailed
	}
	return nil
}

// Test verifies PSRAM is working by writing and reading test patterns.
// This should be called after Init() to verify PSRAM is accessible.
//
// Returns nil on success, or an error if the test fails.
func Test() error {
	if psramTestC() != 0 {
		return ErrTestFailed
	}
	return nil
}

// ErrInitFailed indicates PSRAM initialization failed.
type initError struct{}

func (initError) Error() string { return "psram: initialization failed" }

// ErrTestFailed indicates PSRAM read/write test failed.
type testError struct{}

func (testError) Error() string { return "psram: memory test failed" }

var (
	ErrInitFailed = initError{}
	ErrTestFailed = testError{}
)
