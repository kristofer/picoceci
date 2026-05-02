//go:build tinygo

package tinygo

import (
	"machine"
	"strings"
	"time"
)

// serialConsole wraps machine.Serial for USB CDC on ESP32-S3.
// machine.Serial is the default serial interface, which is USB CDC
// when connected via USB on boards with native USB support.
type serialConsole struct{}

// newConsole creates a console using the default serial interface.
// On ESP32-S3 with USB, this is the USB CDC interface.
func newConsole() Console {
	// machine.Serial is already configured by TinyGo runtime
	return &serialConsole{}
}

func (c *serialConsole) Read(buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		if machine.Serial.Buffered() == 0 {
			if n > 0 {
				break // Return what we have
			}
			// Block until at least one byte is available, with a small yield
			for machine.Serial.Buffered() == 0 {
				time.Sleep(time.Millisecond)
			}
		}
		b, err := machine.Serial.ReadByte()
		if err != nil {
			return n, err
		}
		buf[n] = b
		n++
	}
	return n, nil
}

func (c *serialConsole) Write(buf []byte) (int, error) {
	for _, b := range buf {
		machine.Serial.WriteByte(b)
	}
	return len(buf), nil
}

func (c *serialConsole) ReadLine() (string, error) {
	var sb strings.Builder
	for {
		// Wait for input with a small yield to prevent tight spinning
		for machine.Serial.Buffered() == 0 {
			time.Sleep(time.Millisecond)
		}

		b, err := machine.Serial.ReadByte()
		if err != nil {
			return sb.String(), err
		}

		// Echo the character
		machine.Serial.WriteByte(b)

		if b == '\n' || b == '\r' {
			// Print newline if we got \r
			if b == '\r' {
				machine.Serial.WriteByte('\n')
			}
			break
		}
		// Handle backspace
		if b == 8 || b == 127 {
			s := sb.String()
			if len(s) > 0 {
				sb.Reset()
				sb.WriteString(s[:len(s)-1])
				// Erase character on terminal
				machine.Serial.WriteByte(8)
				machine.Serial.WriteByte(' ')
				machine.Serial.WriteByte(8)
			}
			continue
		}
		sb.WriteByte(b)
	}
	return sb.String(), nil
}

func (c *serialConsole) Available() int {
	return machine.Serial.Buffered()
}
