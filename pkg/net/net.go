package net

// WifiState represents the WiFi station connection state.
type WifiState int

const (
	// WifiStateIdle means the WiFi radio is idle (not connected, not connecting).
	WifiStateIdle WifiState = iota
	// WifiStateConnecting means a connection attempt is in progress.
	WifiStateConnecting
	// WifiStateConnected means the station is associated with an AP.
	WifiStateConnected
	// WifiStateError means the last connection attempt failed.
	WifiStateError
)

// String returns a human-readable name for the state.
func (s WifiState) String() string {
	switch s {
	case WifiStateIdle:
		return "idle"
	case WifiStateConnecting:
		return "connecting"
	case WifiStateConnected:
		return "connected"
	case WifiStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Session represents a single client TCP connection.
// It provides line-oriented input and raw byte output.
type Session interface {
	// Read reads raw bytes from the connection.
	Read(p []byte) (int, error)

	// Write writes raw bytes to the connection.
	Write(p []byte) (int, error)

	// ReadLine reads one text line from the connection.
	// The returned string does not include the trailing newline.
	ReadLine() (string, error)

	// RemoteAddr returns a human-readable string identifying the client address.
	RemoteAddr() string

	// Close closes the connection.
	Close() error
}

// Listener accepts incoming TCP connections.
type Listener interface {
	// Accept waits for the next incoming connection and returns a Session.
	Accept() (Session, error)

	// Addr returns the local listen address (e.g. ":2323").
	Addr() string

	// Close stops accepting new connections.
	Close() error
}

// Manager manages WiFi connectivity and TCP listeners.
// Use NewManager to obtain a platform-appropriate instance.
type Manager struct {
	impl managerImpl
}

// NewManager returns a platform-appropriate Manager.
// On TinyGo targets this wraps the WiFi radio; on desktop it is a no-op stub
// that allows opening regular TCP listeners for testing.
func NewManager() *Manager {
	return &Manager{impl: newManagerImpl()}
}

// Connect connects to the given WiFi network using WPA2 credentials.
// On desktop this is a no-op and always succeeds.
func (m *Manager) Connect(ssid, password string) error {
	return m.impl.connect(ssid, password)
}

// Status returns the current WiFi connection state.
func (m *Manager) Status() WifiState {
	return m.impl.status()
}

// IPAddress returns the current IPv4 address or an empty string when not connected.
func (m *Manager) IPAddress() string {
	return m.impl.ipAddress()
}

// Listen opens a TCP listener on the given port and returns a Listener.
// On desktop this calls net.Listen("tcp", ":port").
// On TinyGo this opens a WiFi socket listener.
func (m *Manager) Listen(port int) (Listener, error) {
	return m.impl.listen(port)
}

// Disconnect disconnects from WiFi and releases all listener sockets.
func (m *Manager) Disconnect() error {
	return m.impl.disconnect()
}

// managerImpl is the platform-specific backend for Manager.
type managerImpl interface {
	connect(ssid, password string) error
	status() WifiState
	ipAddress() string
	listen(port int) (Listener, error)
	disconnect() error
}
