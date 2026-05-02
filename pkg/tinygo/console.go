package tinygo

// Console provides platform-independent console I/O.
// On TinyGo/ESP32, this wraps UART0. On desktop, it wraps stdin/stdout.
type Console interface {
	// Read reads up to len(buf) bytes into buf.
	// Returns the number of bytes read and any error.
	Read(buf []byte) (int, error)

	// Write writes buf to the console.
	// Returns the number of bytes written and any error.
	Write(buf []byte) (int, error)

	// ReadLine reads a single line from the console (blocking).
	// The returned string does not include the trailing newline.
	ReadLine() (string, error)

	// Available returns the number of bytes available to read without blocking.
	Available() int
}

// NewConsole returns a platform-appropriate Console instance.
// On TinyGo, this configures and returns UART0 at 115200 baud.
// On desktop, this returns a stdin/stdout wrapper.
func NewConsole() Console {
	return newConsole()
}
