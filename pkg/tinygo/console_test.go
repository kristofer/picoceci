package tinygo_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kristofer/picoceci/pkg/tinygo"
)

func TestConsoleReadLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple line", "hello\n", "hello"},
		{"with carriage return", "world\r\n", "world"},
		{"empty line", "\n", ""},
		{"with spaces", "hello world\n", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			w := &bytes.Buffer{}
			c := tinygo.NewTestConsole(r, w)

			line, err := c.ReadLine()
			if err != nil {
				t.Fatalf("ReadLine error: %v", err)
			}
			if line != tt.expected {
				t.Errorf("got %q, want %q", line, tt.expected)
			}
		})
	}
}

func TestConsoleWrite(t *testing.T) {
	r := strings.NewReader("")
	w := &bytes.Buffer{}
	c := tinygo.NewTestConsole(r, w)

	msg := []byte("hello picoceci")
	n, err := c.Write(msg)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(msg) {
		t.Errorf("wrote %d bytes, want %d", n, len(msg))
	}
	if w.String() != "hello picoceci" {
		t.Errorf("got %q, want %q", w.String(), "hello picoceci")
	}
}

func TestConsoleRead(t *testing.T) {
	input := "test data"
	r := strings.NewReader(input)
	w := &bytes.Buffer{}
	c := tinygo.NewTestConsole(r, w)

	buf := make([]byte, 4)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if n != 4 {
		t.Errorf("read %d bytes, want 4", n)
	}
	if string(buf) != "test" {
		t.Errorf("got %q, want %q", string(buf), "test")
	}
}

func TestConsoleAvailable(t *testing.T) {
	input := "hello"
	r := strings.NewReader(input)
	w := &bytes.Buffer{}
	c := tinygo.NewTestConsole(r, w)

	// Read one byte to populate the buffer
	buf := make([]byte, 1)
	c.Read(buf)

	// The remaining bytes should be buffered
	available := c.Available()
	if available != 4 {
		t.Errorf("available = %d, want 4", available)
	}
}

func TestConsoleMultipleLines(t *testing.T) {
	input := "line1\nline2\nline3\n"
	r := strings.NewReader(input)
	w := &bytes.Buffer{}
	c := tinygo.NewTestConsole(r, w)

	expected := []string{"line1", "line2", "line3"}
	for i, exp := range expected {
		line, err := c.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine %d error: %v", i, err)
		}
		if line != exp {
			t.Errorf("line %d: got %q, want %q", i, line, exp)
		}
	}
}
