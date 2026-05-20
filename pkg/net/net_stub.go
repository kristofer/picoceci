//go:build !tinygo

package net

import (
	"bufio"
	"fmt"
	stdnet "net"
	"strings"
	"sync"
)

// desktopManagerImpl is the non-TinyGo (desktop/test) backend for Manager.
// WiFi operations are no-ops; TCP listening uses the standard library.
type desktopManagerImpl struct {
	mu        sync.Mutex
	connected bool
	listeners map[*tcpListener]struct{}
}

func newManagerImpl() managerImpl {
	return &desktopManagerImpl{listeners: make(map[*tcpListener]struct{})}
}

func (d *desktopManagerImpl) connect(ssid, password string) error {
	// On desktop, WiFi "connect" is a no-op — we are already on the network.
	d.mu.Lock()
	d.connected = true
	d.mu.Unlock()
	return nil
}

func (d *desktopManagerImpl) status() WifiState {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.connected {
		return WifiStateConnected
	}
	return WifiStateIdle
}

func (d *desktopManagerImpl) ipAddress() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.connected {
		return ""
	}
	return "127.0.0.1"
}

func (d *desktopManagerImpl) listen(port int) (Listener, error) {
	addr := fmt.Sprintf(":%d", port)
	l, err := stdnet.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	tl := &tcpListener{ln: l, owner: d}
	d.mu.Lock()
	d.listeners[tl] = struct{}{}
	d.mu.Unlock()
	return tl, nil
}

func (d *desktopManagerImpl) disconnect() error {
	d.mu.Lock()
	listeners := make([]*tcpListener, 0, len(d.listeners))
	for l := range d.listeners {
		listeners = append(listeners, l)
	}
	d.listeners = make(map[*tcpListener]struct{})
	d.connected = false
	d.mu.Unlock()

	for _, l := range listeners {
		_ = l.Close()
	}
	return nil
}

func (d *desktopManagerImpl) releaseListener(l *tcpListener) {
	d.mu.Lock()
	delete(d.listeners, l)
	d.mu.Unlock()
}

// tcpListener wraps a net.Listener as a picoceci Listener.
type tcpListener struct {
	ln    stdnet.Listener
	owner *desktopManagerImpl

	mu     sync.Mutex
	closed bool
}

func (l *tcpListener) Accept() (Session, error) {
	conn, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return &tcpSession{conn: conn, reader: bufio.NewReader(conn)}, nil
}

func (l *tcpListener) Addr() string {
	return l.ln.Addr().String()
}

func (l *tcpListener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	if l.owner != nil {
		l.owner.releaseListener(l)
	}
	return l.ln.Close()
}

// tcpSession wraps a net.Conn as a picoceci Session.
type tcpSession struct {
	conn   stdnet.Conn
	reader *bufio.Reader
}

func (s *tcpSession) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *tcpSession) Write(p []byte) (int, error) {
	return s.conn.Write(p)
}

func (s *tcpSession) ReadLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func (s *tcpSession) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *tcpSession) Close() error {
	return s.conn.Close()
}
