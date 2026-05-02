package freertos_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/kristofer/picoceci/pkg/freertos"
)

func TestTimerAfter(t *testing.T) {
	var fired int32

	timer := freertos.TimerAfter(50, func() {
		atomic.StoreInt32(&fired, 1)
	})
	defer timer.Delete()

	// Should not have fired yet
	if atomic.LoadInt32(&fired) != 0 {
		t.Error("timer fired immediately")
	}

	// Wait for timer
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&fired) != 1 {
		t.Error("timer did not fire")
	}

	// One-shot should not be active after firing
	if timer.IsActive() {
		t.Error("one-shot timer still active after firing")
	}
}

func TestTimerEvery(t *testing.T) {
	var count int32

	timer := freertos.TimerEvery(30, func() {
		atomic.AddInt32(&count, 1)
	})
	defer timer.Delete()

	// Wait for multiple fires
	time.Sleep(150 * time.Millisecond)

	// Should have fired multiple times
	c := atomic.LoadInt32(&count)
	if c < 3 {
		t.Errorf("periodic timer fired %d times, want >= 3", c)
	}

	// Periodic should still be active
	if !timer.IsActive() {
		t.Error("periodic timer not active")
	}
}

func TestTimerStop(t *testing.T) {
	var count int32

	timer := freertos.TimerEvery(20, func() {
		atomic.AddInt32(&count, 1)
	})

	// Wait for some fires
	time.Sleep(60 * time.Millisecond)

	countAtStop := atomic.LoadInt32(&count)

	// Stop the timer
	if !timer.Stop() {
		t.Error("Stop failed")
	}

	// Wait more
	time.Sleep(60 * time.Millisecond)

	// Count should not have increased much (maybe one in-flight)
	finalCount := atomic.LoadInt32(&count)
	if finalCount > countAtStop+1 {
		t.Errorf("timer continued firing after Stop: %d -> %d", countAtStop, finalCount)
	}

	timer.Delete()
}

func TestTimerRestart(t *testing.T) {
	var count int32

	timer := freertos.TimerEvery(30, func() {
		atomic.AddInt32(&count, 1)
	})

	// Wait for some fires
	time.Sleep(80 * time.Millisecond)

	timer.Stop()
	countAtStop := atomic.LoadInt32(&count)

	// Restart
	time.Sleep(50 * time.Millisecond)
	timer.Start()

	// Wait for more fires
	time.Sleep(80 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount <= countAtStop {
		t.Errorf("timer did not restart: count stayed at %d", countAtStop)
	}

	timer.Delete()
}

func TestTimerIsPeriodic(t *testing.T) {
	oneshot := freertos.TimerAfter(100, func() {})
	defer oneshot.Delete()

	periodic := freertos.TimerEvery(100, func() {})
	defer periodic.Delete()

	if oneshot.IsPeriodic() {
		t.Error("one-shot timer IsPeriodic = true")
	}

	if !periodic.IsPeriodic() {
		t.Error("periodic timer IsPeriodic = false")
	}
}

func TestTimerPeriod(t *testing.T) {
	timer := freertos.TimerAfter(123, func() {})
	defer timer.Delete()

	if timer.Period() != 123 {
		t.Errorf("Period = %d, want 123", timer.Period())
	}
}

func TestTimerReset(t *testing.T) {
	var fired int32

	timer := freertos.TimerAfter(200, func() {
		atomic.StoreInt32(&fired, 1)
	})
	defer timer.Delete()

	// Reset to shorter period
	timer.Reset(30)

	// Should fire sooner now
	time.Sleep(60 * time.Millisecond)

	if atomic.LoadInt32(&fired) != 1 {
		t.Error("timer did not fire after Reset to shorter period")
	}
}

func TestTimerDelete(t *testing.T) {
	var count int32

	timer := freertos.TimerEvery(20, func() {
		atomic.AddInt32(&count, 1)
	})

	time.Sleep(50 * time.Millisecond)
	timer.Delete()

	countAtDelete := atomic.LoadInt32(&count)

	time.Sleep(50 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount > countAtDelete+1 {
		t.Errorf("timer continued after Delete: %d -> %d", countAtDelete, finalCount)
	}

	// Operations on deleted timer should fail
	if timer.Start() {
		t.Error("Start on deleted timer succeeded")
	}
	if timer.IsActive() {
		t.Error("deleted timer IsActive = true")
	}
}

func TestTimerIsActive(t *testing.T) {
	timer := freertos.TimerEvery(50, func() {})

	if !timer.IsActive() {
		t.Error("new timer IsActive = false")
	}

	timer.Stop()

	if timer.IsActive() {
		t.Error("stopped timer IsActive = true")
	}

	timer.Delete()
}
