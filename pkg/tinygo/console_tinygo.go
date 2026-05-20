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

const maxSerialLineBytes = 1024

// newConsole creates a console using the default serial interface.
// On ESP32-S3 with USB, this is the USB CDC interface.
func newConsole() Console {
	// machine.Serial is already configured by TinyGo runtime
	return &serialConsole{}
}

func (c *serialConsole) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

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
	line := make([]byte, 0, 64)
	for {
		// Wait for input with a small yield to prevent tight spinning
		for machine.Serial.Buffered() == 0 {
			time.Sleep(time.Millisecond)
		}

		b, err := machine.Serial.ReadByte()
		if err != nil {
			return string(line), err
		}

		switch b {
		case '\n', '\r':
			machine.Serial.WriteByte('\r')
			machine.Serial.WriteByte('\n')
			return strings.TrimRight(string(line), "\r\n"), nil
		case 8, 127:
			if len(line) > 0 {
				line = line[:len(line)-1]
				machine.Serial.WriteByte(8)
				machine.Serial.WriteByte(' ')
				machine.Serial.WriteByte(8)
			}
			continue
		default:
			if b < 32 && b != '\t' {
				continue
			}
			if len(line) >= maxSerialLineBytes {
				// Bell feedback warns that further input is ignored for this line.
				machine.Serial.WriteByte('\a')
				continue
			}
			line = append(line, b)
			machine.Serial.WriteByte(b)
		}
	}
}

func (c *serialConsole) Available() int {
	return machine.Serial.Buffered()
}
