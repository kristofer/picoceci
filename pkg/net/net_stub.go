//go:build !tinygo

package net

import (
	"bufio"
	"fmt"
	stdnet "net"
	"strings"
)

// desktopManagerImpl is the non-TinyGo (desktop/test) backend for Manager.
// WiFi operations are no-ops; TCP listening uses the standard library.
type desktopManagerImpl struct {
	connected bool
}

func newManagerImpl() managerImpl {
	return &desktopManagerImpl{}
}

func (d *desktopManagerImpl) connect(ssid, password string) error {
	// On desktop, WiFi "connect" is a no-op — we are already on the network.
	d.connected = true
	return nil
}

func (d *desktopManagerImpl) status() WifiState {
	if d.connected {
		return WifiStateConnected
	}
	return WifiStateIdle
}

func (d *desktopManagerImpl) ipAddress() string {
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
	return &tcpListener{ln: l}, nil
}

func (d *desktopManagerImpl) disconnect() error {
	d.connected = false
	return nil
}

// tcpListener wraps a net.Listener as a picoceci Listener.
type tcpListener struct {
	ln stdnet.Listener
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
	if err != nil {
		return strings.TrimRight(line, "\r\n"), err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (s *tcpSession) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *tcpSession) Close() error {
	return s.conn.Close()
}
