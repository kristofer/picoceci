//go:build tinygo

package tinygo

import (
	"machine"
	"strings"
)

// uartConsole wraps machine.UART0 for ESP32-S3.
type uartConsole struct {
	uart *machine.UART
	buf  []byte
}

// newConsole creates a UART0 console at 115200 baud.
func newConsole() Console {
	uart := machine.UART0
	uart.Configure(machine.UARTConfig{BaudRate: 115200})
	return &uartConsole{
		uart: uart,
		buf:  make([]byte, 256),
	}
}

func (c *uartConsole) Read(buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		if c.uart.Buffered() == 0 {
			if n > 0 {
				break // Return what we have
			}
			// Block until at least one byte is available
			for c.uart.Buffered() == 0 {
				// busy wait
			}
		}
		b, err := c.uart.ReadByte()
		if err != nil {
			return n, err
		}
		buf[n] = b
		n++
	}
	return n, nil
}

func (c *uartConsole) Write(buf []byte) (int, error) {
	return c.uart.Write(buf)
}

func (c *uartConsole) ReadLine() (string, error) {
	var sb strings.Builder
	for {
		if c.uart.Buffered() == 0 {
			// busy wait for input
			for c.uart.Buffered() == 0 {
			}
		}
		b, err := c.uart.ReadByte()
		if err != nil {
			return sb.String(), err
		}
		// Echo the character
		c.uart.WriteByte(b)

		if b == '\n' || b == '\r' {
			// Print newline if we got \r
			if b == '\r' {
				c.uart.WriteByte('\n')
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
				c.uart.Write([]byte{8, ' ', 8})
			}
			continue
		}
		sb.WriteByte(b)
	}
	return sb.String(), nil
}

func (c *uartConsole) Available() int {
	return c.uart.Buffered()
}
