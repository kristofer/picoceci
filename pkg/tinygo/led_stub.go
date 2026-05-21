//go:build !tinygo

package tinygo

import (
	"sync"
	"time"
)

// stubLED is a desktop LED stub that tracks on/off state and blink timer.
type stubLED struct {
	mu   sync.Mutex
	on   bool
	stop chan struct{}
}

func newLED() LED {
	return &stubLED{}
}

func (l *stubLED) On() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.on = true
}

func (l *stubLED) Off() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.on = false
}

func (l *stubLED) Toggle() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.on = !l.on
}

func (l *stubLED) BlinkEvery(ms int) {
	l.mu.Lock()
	l.stopBlinkLocked()
	stop := make(chan struct{})
	l.stop = stop
	l.mu.Unlock()

	go func() {
		ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				l.Toggle()
			}
		}
	}()
}

func (l *stubLED) StopBlink() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopBlinkLocked()
}

// stopBlinkLocked stops any running blink goroutine. Must be called with l.mu held.
func (l *stubLED) stopBlinkLocked() {
	if l.stop != nil {
		close(l.stop)
		l.stop = nil
	}
}
