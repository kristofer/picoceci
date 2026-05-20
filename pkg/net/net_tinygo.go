//go:build tinygo && !esp32s3_idf_bridge

package net

import (
	"bufio"
	"fmt"
	stdnet "net"
	"strings"
	"sync"
)

// tinygoManagerImpl is the TinyGo backend for WiFi/TCP access.
// It manages connection state and wraps TCP sockets in picoceci Session/Listener
// interfaces.
type tinygoManagerImpl struct {
	mu        sync.Mutex
	state     WifiState
	ip        string
	listeners map[*tinygoListener]struct{}
}

func newManagerImpl() managerImpl {
	return &tinygoManagerImpl{
		state:     WifiStateIdle,
		listeners: make(map[*tinygoListener]struct{}),
	}
}

func (t *tinygoManagerImpl) connect(ssid, password string) error {
	if strings.TrimSpace(ssid) == "" {
		t.mu.Lock()
		t.state = WifiStateError
		t.mu.Unlock()
		return fmt.Errorf("wifi ssid must not be empty")
	}

	t.mu.Lock()
	t.state = WifiStateConnecting
	t.mu.Unlock()

	if err := joinWiFiNetwork(ssid, password); err != nil {
		t.mu.Lock()
		t.state = WifiStateError
		t.mu.Unlock()
		return err
	}

	t.mu.Lock()
	t.state = WifiStateConnected
	if t.ip == "" {
		t.ip = "0.0.0.0"
	}
	t.mu.Unlock()

	return nil
}

func (t *tinygoManagerImpl) status() WifiState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

func (t *tinygoManagerImpl) ipAddress() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ip
}

func (t *tinygoManagerImpl) listen(port int) (Listener, error) {
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port %d", port)
	}

	t.mu.Lock()
	connected := t.state == WifiStateConnected
	t.mu.Unlock()
	if !connected {
		return nil, fmt.Errorf("wifi is not connected")
	}

	ln, err := stdnet.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.mu.Lock()
		t.state = WifiStateError
		t.mu.Unlock()
		return nil, err
	}

	tl := &tinygoListener{ln: ln, owner: t}
	t.mu.Lock()
	t.listeners[tl] = struct{}{}
	t.mu.Unlock()

	if hostIP := localIPv4FromAddr(ln.Addr()); hostIP != "" {
		t.mu.Lock()
		t.ip = hostIP
		t.mu.Unlock()
	}

	return tl, nil
}

func (t *tinygoManagerImpl) disconnect() error {
	t.mu.Lock()
	listeners := make([]*tinygoListener, 0, len(t.listeners))
	for l := range t.listeners {
		listeners = append(listeners, l)
	}
	t.listeners = make(map[*tinygoListener]struct{})
	t.state = WifiStateIdle
	t.ip = ""
	t.mu.Unlock()

	for _, l := range listeners {
		_ = l.Close()
	}

	return nil
}

func (t *tinygoManagerImpl) releaseListener(l *tinygoListener) {
	t.mu.Lock()
	delete(t.listeners, l)
	t.mu.Unlock()
}

// joinWiFiNetwork associates the station with an AP.
// This is kept as an explicit hook point so board-specific ESP32/TinyGo WiFi
// association code can be dropped in without changing Manager behavior.
func joinWiFiNetwork(ssid, password string) error {
	_ = ssid
	_ = password
	return nil
}

func localIPv4FromAddr(addr stdnet.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := stdnet.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return ""
	}
	return host
}

type tinygoListener struct {
	ln    stdnet.Listener
	owner *tinygoManagerImpl

	mu     sync.Mutex
	closed bool
}

func (l *tinygoListener) Accept() (Session, error) {
	conn, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return &tinygoSession{conn: conn, reader: bufio.NewReader(conn)}, nil
}

func (l *tinygoListener) Addr() string {
	return l.ln.Addr().String()
}

func (l *tinygoListener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	l.owner.releaseListener(l)
	return l.ln.Close()
}

type tinygoSession struct {
	conn   stdnet.Conn
	reader *bufio.Reader
}

func (s *tinygoSession) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *tinygoSession) Write(p []byte) (int, error) {
	return s.conn.Write(p)
}

func (s *tinygoSession) ReadLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func (s *tinygoSession) RemoteAddr() string {
	if s.conn == nil || s.conn.RemoteAddr() == nil {
		return ""
	}
	return s.conn.RemoteAddr().String()
}

func (s *tinygoSession) Close() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}
