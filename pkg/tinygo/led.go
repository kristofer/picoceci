package tinygo

// LED is the platform-independent LED driver interface.
// On TinyGo/ESP32-S3, this drives machine.LED.
// On desktop, it is a state-tracking no-op stub.
type LED interface {
	// On sets the LED high.
	// For RGB LEDs, this restores the current color or a board default.
	On()
	// Off sets the LED low.
	Off()
	// Toggle inverts the current LED state.
	Toggle()
	// Red sets the LED to red and turns it on.
	Red()
	// Green sets the LED to green and turns it on.
	Green()
	// Blue sets the LED to blue and turns it on.
	Blue()
	// White sets the LED to white and turns it on.
	White()
	// RGB sets an explicit RGB color and turns the LED on.
	RGB(r, g, b uint8)
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
