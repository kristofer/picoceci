package tinygo

// LED is the platform-independent LED driver interface.
// On TinyGo/ESP32-S3, this drives machine.LED.
// On desktop, it is a state-tracking no-op stub.
type LED interface {
	// On sets the LED high.
	On()
	// Off sets the LED low.
	Off()
	// Toggle inverts the current LED state.
	Toggle()
	// BlinkEvery starts a periodic blink at the given millisecond interval.
	// Replaces any previously running blink timer.
	BlinkEvery(ms int)
	// StopBlink stops any running blink timer.
	StopBlink()
}

// NewLED returns a platform-appropriate LED instance.
// On TinyGo/ESP32-S3, this configures and returns a machine.LED driver.
// On desktop, this returns a stub that tracks state without hardware I/O.
func NewLED() LED {
	return newLED()
}
