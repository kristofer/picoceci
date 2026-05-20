//go:build tinygo && esp32s3_idf_bridge

package net

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
)

//export picoceci_bridge_wifi_stack_init
func bridgeWiFiStackInit() int32

//export picoceci_bridge_wifi_connect
func bridgeWiFiConnect(ssid *byte, password *byte, timeoutMs uint32, outIP *uint32) int32

//export picoceci_bridge_wifi_disconnect
func bridgeWiFiDisconnect() int32

//export picoceci_bridge_tcp_listen
func bridgeTCPListen(port uint16) int32

//export picoceci_bridge_tcp_accept
func bridgeTCPAccept(serverFD int32, outIP *uint32, outPort *uint16) int32

//export picoceci_bridge_tcp_recv
func bridgeTCPRecv(fd int32, buf *byte, n uint32) int32

//export picoceci_bridge_tcp_send
func bridgeTCPSend(fd int32, buf *byte, n uint32) int32

//export picoceci_bridge_tcp_close
func bridgeTCPClose(fd int32) int32

// Bridge return-code groups, kept centralized to make Phase 2 error mapping explicit.
const (
	bridgeWiFiInitMin int32 = -6
	bridgeWiFiInitMax int32 = -1

	bridgeWiFiConnectMin int32 = -15
	bridgeWiFiConnectMax int32 = -10

	bridgeWiFiDisconnectMin int32 = -21
	bridgeWiFiDisconnectMax int32 = -20

	bridgeTCPListenMin int32 = -32
	bridgeTCPListenMax int32 = -30

	bridgeTCPAcceptRC int32 = -40
)

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

	if rc := bridgeWiFiStackInit(); rc != 0 {
		t.mu.Lock()
		t.state = WifiStateError
		t.mu.Unlock()
		return bridgeError("wifi stack init", rc)
	}

	ssidBuf := append([]byte(ssid), 0)
	ssidPtr := &ssidBuf[0]
	var passPtr *byte
	passBuf := append([]byte(password), 0)
	if password != "" {
		passPtr = &passBuf[0]
	}

	var ip uint32
	if rc := bridgeWiFiConnect(ssidPtr, passPtr, 15000, &ip); rc != 0 {
		t.mu.Lock()
		t.state = WifiStateError
		t.mu.Unlock()
		return bridgeError("wifi connect", rc)
	}

	t.mu.Lock()
	t.state = WifiStateConnected
	t.ip = ipToString(ip)
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

	fd := bridgeTCPListen(uint16(port))
	if fd < 0 {
		t.mu.Lock()
		t.state = WifiStateError
		t.mu.Unlock()
		return nil, bridgeError("tcp listen", fd)
	}

	tl := &tinygoListener{fd: fd, port: port, owner: t}
	t.mu.Lock()
	t.listeners[tl] = struct{}{}
	t.mu.Unlock()
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
	if rc := bridgeWiFiDisconnect(); rc != 0 {
		return bridgeError("wifi disconnect", rc)
	}
	return nil
}

func (t *tinygoManagerImpl) releaseListener(l *tinygoListener) {
	t.mu.Lock()
	delete(t.listeners, l)
	t.mu.Unlock()
}

type tinygoListener struct {
	fd    int32
	port  int
	owner *tinygoManagerImpl

	mu     sync.Mutex
	closed bool
}

func (l *tinygoListener) Accept() (Session, error) {
	l.mu.Lock()
	closed := l.closed
	l.mu.Unlock()
	if closed {
		return nil, fmt.Errorf("listener is closed")
	}

	var ip uint32
	var port uint16
	cfd := bridgeTCPAccept(l.fd, &ip, &port)
	if cfd < 0 {
		return nil, bridgeError("tcp accept", cfd)
	}

	s := &tinygoSession{
		fd:         cfd,
		remoteAddr: fmt.Sprintf("%s:%d", ipToString(ip), port),
	}
	s.reader = bufio.NewReader(s)
	return s, nil
}

func (l *tinygoListener) Addr() string {
	return fmt.Sprintf(":%d", l.port)
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
	if rc := bridgeTCPClose(l.fd); rc != 0 {
		return bridgeError("tcp close(listener)", rc)
	}
	return nil
}

type tinygoSession struct {
	fd         int32
	remoteAddr string
	reader     *bufio.Reader
}

func (s *tinygoSession) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := bridgeTCPRecv(s.fd, &p[0], uint32(len(p)))
	if n < 0 {
		return 0, bridgeError("tcp recv", n)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return int(n), nil
}

func (s *tinygoSession) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := bridgeTCPSend(s.fd, &p[0], uint32(len(p)))
	if n <= 0 {
		if n == 0 {
			return 0, io.ErrShortWrite
		}
		return 0, bridgeError("tcp send", n)
	}
	return int(n), nil
}

func (s *tinygoSession) ReadLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func (s *tinygoSession) RemoteAddr() string {
	return s.remoteAddr
}

func (s *tinygoSession) Close() error {
	if rc := bridgeTCPClose(s.fd); rc != 0 {
		return bridgeError("tcp close(session)", rc)
	}
	return nil
}

func bridgeError(op string, rc int32) error {
	return fmt.Errorf("bridge %s failed (rc=%d, class=%s)", op, rc, bridgeRCClass(rc))
}

func bridgeRCClass(rc int32) string {
	switch {
	case rc >= bridgeWiFiInitMin && rc <= bridgeWiFiInitMax:
		return "wifi-init"
	case rc >= bridgeWiFiConnectMin && rc <= bridgeWiFiConnectMax:
		return "wifi-connect"
	case rc >= bridgeWiFiDisconnectMin && rc <= bridgeWiFiDisconnectMax:
		return "wifi-disconnect"
	case rc >= bridgeTCPListenMin && rc <= bridgeTCPListenMax:
		return "tcp-listen"
	case rc == bridgeTCPAcceptRC:
		return "tcp-accept"
	case rc < 0:
		return "tcp-socket"
	default:
		return "non-error"
	}
}

func ipToString(ip uint32) string {
	a := (ip >> 24) & 0xFF
	b := (ip >> 16) & 0xFF
	c := (ip >> 8) & 0xFF
	d := ip & 0xFF
	return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
}
