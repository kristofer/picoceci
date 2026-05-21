//go:build tinygo && !esp32s3_n16r8

package tinygo

import (
	"machine"
	"time"
)

// boardLED drives a board LED pin when available.
type boardLED struct {
	pin  machine.Pin
	stop chan struct{}
}

func newLED() LED {
	// Targets without an explicit board pin mapping use NoPin as a safe no-op fallback.
	p := machine.NoPin
	if p != machine.NoPin {
		p.Configure(machine.PinConfig{Mode: machine.PinOutput})
	}
	return &boardLED{pin: p}
}

func (l *boardLED) On() {
	if l.pin == machine.NoPin {
		return
	}
	l.pin.High()
}

func (l *boardLED) Off() {
	if l.pin == machine.NoPin {
		return
	}
	l.pin.Low()
}

func (l *boardLED) Toggle() {
	if l.pin == machine.NoPin {
		return
	}
	l.pin.Set(!l.pin.Get())
}

func (l *boardLED) BlinkEvery(ms int) {
	if l.stop != nil {
		close(l.stop)
	}
	stop := make(chan struct{})
	l.stop = stop
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Duration(ms) * time.Millisecond):
				l.Toggle()
			}
		}
	}()
}

func (l *boardLED) StopBlink() {
	if l.stop != nil {
		close(l.stop)
		l.stop = nil
	}
}
