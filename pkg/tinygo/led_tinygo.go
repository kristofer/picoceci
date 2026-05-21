//go:build tinygo

package tinygo

import (
	"machine"
	"time"
)

// boardLED drives machine.LED on the ESP32-S3 board.
type boardLED struct {
	pin  machine.Pin
	stop chan struct{}
}

func newLED() LED {
	p := machine.LED
	p.Configure(machine.PinConfig{Mode: machine.PinOutput})
	return &boardLED{pin: p}
}

func (l *boardLED) On() {
	l.pin.High()
}

func (l *boardLED) Off() {
	l.pin.Low()
}

func (l *boardLED) Toggle() {
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
