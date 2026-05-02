//go:build !tinygo

package tinygo

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// stdioConsole wraps stdin/stdout for desktop testing.
type stdioConsole struct {
	reader *bufio.Reader
	writer io.Writer
}

// newConsole creates a desktop console using stdin/stdout.
func newConsole() Console {
	return &stdioConsole{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// NewTestConsole creates a console with custom reader/writer for testing.
func NewTestConsole(r io.Reader, w io.Writer) Console {
	return &stdioConsole{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

func (c *stdioConsole) Read(buf []byte) (int, error) {
	return c.reader.Read(buf)
}

func (c *stdioConsole) Write(buf []byte) (int, error) {
	return c.writer.Write(buf)
}

func (c *stdioConsole) ReadLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return line, err
	}
	// Strip trailing newline (and carriage return if present)
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

func (c *stdioConsole) Available() int {
	return c.reader.Buffered()
}
