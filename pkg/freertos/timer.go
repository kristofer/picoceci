package freertos

// Timer provides scheduled callback execution.
// On FreeRTOS, this wraps xTimerCreate/xTimerStart/xTimerStop.
// On desktop, this uses time.AfterFunc/time.Ticker.
type Timer interface {
	// Start starts or restarts the timer.
	Start() bool

	// Stop stops the timer.
	Stop() bool

	// Reset resets the timer with a new period.
	Reset(periodMs uint32) bool

	// IsActive returns true if the timer is running.
	IsActive() bool

	// IsPeriodic returns true if the timer repeats.
	IsPeriodic() bool

	// Period returns the timer period in milliseconds.
	Period() uint32

	// Delete releases timer resources.
	Delete()
}

// TimerFunc is a function called when the timer fires.
type TimerFunc func()

// TimerAfter creates a one-shot timer that fires once after ms milliseconds.
func TimerAfter(ms uint32, fn TimerFunc) Timer {
	return newTimer(ms, fn, false)
}

// TimerEvery creates a periodic timer that fires every ms milliseconds.
func TimerEvery(ms uint32, fn TimerFunc) Timer {
	return newTimer(ms, fn, true)
}
