//go:build tinygo

package freertos

import (
	"sync"
	"time"
)

// tinygoTimer implements Timer for FreeRTOS.
// This is a placeholder using the same implementation as the stub.
// TODO: Replace with actual FreeRTOS timer calls.
type tinygoTimer struct {
	periodMs uint32
	fn       TimerFunc
	periodic bool
	active   bool
	deleted  bool
	timer    *time.Timer
	ticker   *time.Ticker
	stopCh   chan struct{}
	mu       sync.Mutex
	// handle uintptr  // FreeRTOS TimerHandle_t
}

func newTimer(ms uint32, fn TimerFunc, periodic bool) Timer {
	// TODO: Call xTimerCreate
	t := &tinygoTimer{
		periodMs: ms,
		fn:       fn,
		periodic: periodic,
		stopCh:   make(chan struct{}),
	}
	t.Start()
	return t
}

func (t *tinygoTimer) Start() bool {
	// TODO: Call xTimerStart
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.deleted || t.active {
		return false
	}

	t.active = true

	if t.periodic {
		t.ticker = time.NewTicker(time.Duration(t.periodMs) * time.Millisecond)
		go func() {
			for {
				select {
				case <-t.ticker.C:
					t.mu.Lock()
					if t.deleted || !t.active {
						t.mu.Unlock()
						return
					}
					fn := t.fn
					t.mu.Unlock()
					if fn != nil {
						fn()
					}
				case <-t.stopCh:
					return
				}
			}
		}()
	} else {
		t.timer = time.AfterFunc(time.Duration(t.periodMs)*time.Millisecond, func() {
			t.mu.Lock()
			if t.deleted || !t.active {
				t.mu.Unlock()
				return
			}
			t.active = false
			fn := t.fn
			t.mu.Unlock()
			if fn != nil {
				fn()
			}
		})
	}

	return true
}

func (t *tinygoTimer) Stop() bool {
	// TODO: Call xTimerStop
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.deleted || !t.active {
		return false
	}

	t.active = false

	if t.periodic && t.ticker != nil {
		t.ticker.Stop()
		select {
		case t.stopCh <- struct{}{}:
		default:
		}
	} else if t.timer != nil {
		t.timer.Stop()
	}

	return true
}

func (t *tinygoTimer) Reset(periodMs uint32) bool {
	// TODO: Call xTimerChangePeriod
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.deleted {
		return false
	}

	t.periodMs = periodMs

	if t.active {
		if t.periodic && t.ticker != nil {
			t.ticker.Reset(time.Duration(periodMs) * time.Millisecond)
		} else if t.timer != nil {
			t.timer.Reset(time.Duration(periodMs) * time.Millisecond)
		}
	}

	return true
}

func (t *tinygoTimer) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active && !t.deleted
}

func (t *tinygoTimer) IsPeriodic() bool {
	return t.periodic
}

func (t *tinygoTimer) Period() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.periodMs
}

func (t *tinygoTimer) Delete() {
	// TODO: Call xTimerDelete
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.deleted {
		return
	}

	t.deleted = true
	t.active = false

	if t.periodic && t.ticker != nil {
		t.ticker.Stop()
		close(t.stopCh)
	} else if t.timer != nil {
		t.timer.Stop()
	}
}

// FreeRTOS function declarations (to be implemented with //go:linkname):
//
// //go:linkname xTimerCreate xTimerCreate
// func xTimerCreate(pcTimerName *byte, xTimerPeriodInTicks uint32,
//     uxAutoReload int32, pvTimerID unsafe.Pointer,
//     pxCallbackFunction uintptr) uintptr
//
// //go:linkname xTimerDelete xTimerDelete
// func xTimerDelete(xTimer uintptr, xTicksToWait uint32) int32
//
// //go:linkname xTimerStart xTimerStart
// func xTimerStart(xTimer uintptr, xTicksToWait uint32) int32
//
// //go:linkname xTimerStop xTimerStop
// func xTimerStop(xTimer uintptr, xTicksToWait uint32) int32
//
// //go:linkname xTimerReset xTimerReset
// func xTimerReset(xTimer uintptr, xTicksToWait uint32) int32
//
// //go:linkname xTimerChangePeriod xTimerChangePeriod
// func xTimerChangePeriod(xTimer uintptr, xNewPeriod uint32, xTicksToWait uint32) int32
//
// //go:linkname xTimerIsTimerActive xTimerIsTimerActive
// func xTimerIsTimerActive(xTimer uintptr) int32
