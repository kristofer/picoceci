//go:build !tinygo

package freertos

import (
	"sync"
	"time"
)

// stubTimer implements Timer using time.AfterFunc/time.Ticker.
type stubTimer struct {
	periodMs uint32
	fn       TimerFunc
	periodic bool
	active   bool
	deleted  bool
	timer    *time.Timer
	ticker   *time.Ticker
	stopCh   chan struct{}
	mu       sync.Mutex
}

func newTimer(ms uint32, fn TimerFunc, periodic bool) Timer {
	t := &stubTimer{
		periodMs: ms,
		fn:       fn,
		periodic: periodic,
		stopCh:   make(chan struct{}),
	}
	// Start immediately
	t.Start()
	return t
}

func (t *stubTimer) Start() bool {
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

func (t *stubTimer) Stop() bool {
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

func (t *stubTimer) Reset(periodMs uint32) bool {
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

func (t *stubTimer) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active && !t.deleted
}

func (t *stubTimer) IsPeriodic() bool {
	return t.periodic
}

func (t *stubTimer) Period() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.periodMs
}

func (t *stubTimer) Delete() {
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
