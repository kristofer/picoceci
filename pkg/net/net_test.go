package net_test

import (
	"fmt"
	stdnet "net"
	"strings"
	"testing"
	"time"

	picnet "github.com/kristofer/picoceci/pkg/net"
)

// TestManagerInitialState verifies new managers start idle with no IP.
func TestManagerInitialState(t *testing.T) {
	m := picnet.NewManager()
	if got := m.Status(); got != picnet.WifiStateIdle {
		t.Fatalf("initial Status() = %v, want idle", got)
	}
	if got := m.IPAddress(); got != "" {
		t.Fatalf("initial IPAddress() = %q, want empty", got)
	}
}

// TestManagerConnect verifies that desktop connect is a no-op success.
func TestManagerConnect(t *testing.T) {
	m := picnet.NewManager()
	if err := m.Connect("testnet", "password"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if m.Status() != picnet.WifiStateConnected {
		t.Errorf("Status() = %v, want connected", m.Status())
	}
	if m.IPAddress() == "" {
		t.Error("IPAddress() is empty after connect")
	}
}

// TestManagerDisconnect verifies that Disconnect resets state.
func TestManagerDisconnect(t *testing.T) {
	m := picnet.NewManager()
	_ = m.Connect("testnet", "password")
	if got := m.Status(); got != picnet.WifiStateConnected {
		t.Fatalf("precondition Status() = %v, want connected", got)
	}
	if err := m.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
	if m.Status() != picnet.WifiStateIdle {
		t.Errorf("Status() = %v after disconnect, want idle", m.Status())
	}
	if got := m.IPAddress(); got != "" {
		t.Errorf("IPAddress() = %q after disconnect, want empty", got)
	}
}

// TestListenerCloseUnblocksAccept verifies listener close terminates pending Accept.
func TestListenerCloseUnblocksAccept(t *testing.T) {
	m := picnet.NewManager()
	ln, err := m.Listen(0)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	acceptErr := make(chan error, 1)
	go func() {
		_, err := ln.Accept()
		acceptErr <- err
	}()

	time.Sleep(100 * time.Millisecond)
	if err := ln.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	select {
	case err := <-acceptErr:
		if err == nil {
			t.Fatal("Accept returned nil error after listener close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not unblock after listener close")
	}
}

// TestDisconnectClosesListeners verifies Disconnect tears down active listeners.
func TestDisconnectClosesListeners(t *testing.T) {
	m := picnet.NewManager()
	_ = m.Connect("testnet", "password")

	ln, err := m.Listen(0)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	acceptErr := make(chan error, 1)
	go func() {
		_, err := ln.Accept()
		acceptErr <- err
	}()

	time.Sleep(100 * time.Millisecond)
	if err := m.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	select {
	case err := <-acceptErr:
		if err == nil {
			t.Fatal("Accept returned nil error after manager disconnect")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not unblock after manager disconnect")
	}

	if got := m.Status(); got != picnet.WifiStateIdle {
		t.Fatalf("Status() after disconnect = %v, want idle", got)
	}
}

// TestListenerAcceptSession verifies the listener/session lifecycle:
//   - Start a listener on a random port
//   - Connect with a standard TCP client
//   - Write a line, read the echo back
//   - Close
func TestListenerAcceptSession(t *testing.T) {
	m := picnet.NewManager()
	// Port 0 lets the OS choose an available port.
	ln, err := m.Listen(0)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr()
	if addr == "" {
		t.Fatal("Addr() is empty")
	}

	// Server goroutine: accept one session, echo one line, close.
	serverDone := make(chan error, 1)
	go func() {
		sess, err := ln.Accept()
		if err != nil {
			serverDone <- fmt.Errorf("Accept: %w", err)
			return
		}
		defer sess.Close()

		line, err := sess.ReadLine()
		if err != nil {
			serverDone <- fmt.Errorf("ReadLine: %w", err)
			return
		}
		_, err = sess.Write([]byte(line + "\n"))
		serverDone <- err
	}()

	// Client: connect, send a line, read the echo.
	conn, err := stdnet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	const msg = "hello picoceci"
	fmt.Fprintln(conn, msg)

	buf := make([]byte, 64)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read from conn: %v", err)
	}
	got := strings.TrimRight(string(buf[:n]), "\r\n")
	if got != msg {
		t.Errorf("echo = %q, want %q", got, msg)
	}

	// Wait for server goroutine.
	if err := <-serverDone; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

// TestSessionReadLine verifies that ReadLine strips the newline delimiter.
func TestSessionReadLine(t *testing.T) {
	m := picnet.NewManager()
	ln, err := m.Listen(0)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan string, 1)
	go func() {
		sess, err := ln.Accept()
		if err != nil {
			serverDone <- ""
			return
		}
		defer sess.Close()
		line, _ := sess.ReadLine()
		serverDone <- line
	}()

	conn, err := stdnet.DialTimeout("tcp", ln.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	// Send a line with CRLF (common with nc and telnet clients).
	fmt.Fprint(conn, "ping\r\n")

	got := <-serverDone
	if got != "ping" {
		t.Errorf("ReadLine = %q, want %q", got, "ping")
	}
}

// TestSessionCloseThenRead verifies reads fail after remote close.
func TestSessionCloseThenRead(t *testing.T) {
	m := picnet.NewManager()
	ln, err := m.Listen(0)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan error, 1)
	go func() {
		sess, err := ln.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		_ = sess.Close()

		buf := make([]byte, 1)
		_, err = sess.Read(buf)
		if err == nil {
			serverDone <- fmt.Errorf("expected read error after session close")
			return
		}
		serverDone <- nil
	}()

	conn, err := stdnet.DialTimeout("tcp", ln.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	if err := <-serverDone; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

// TestListenerIntegration_REPLEcho simulates a REPL session:
// the server reads lines of picoceci code, echoes them prefixed with "=> ".
// This mirrors how PicoceciREPL serve: session would behave.
func TestListenerIntegration_REPLEcho(t *testing.T) {
	m := picnet.NewManager()
	ln, err := m.Listen(0)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	const numLines = 3
	serverDone := make(chan error, 1)
	go func() {
		sess, err := ln.Accept()
		if err != nil {
			serverDone <- fmt.Errorf("Accept: %w", err)
			return
		}
		defer sess.Close()
		for i := 0; i < numLines; i++ {
			line, err := sess.ReadLine()
			if err != nil {
				serverDone <- fmt.Errorf("ReadLine %d: %w", i, err)
				return
			}
			_, err = sess.Write([]byte("=> " + line + "\n"))
			if err != nil {
				serverDone <- fmt.Errorf("Write %d: %w", i, err)
				return
			}
		}
		serverDone <- nil
	}()

	conn, err := stdnet.DialTimeout("tcp", ln.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	lines := []string{"42.", "'hello'.", "3 + 4."}
	for i, line := range lines {
		fmt.Fprintln(conn, line)
		buf := make([]byte, 128)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("Read response %d: %v", i, err)
		}
		got := strings.TrimRight(string(buf[:n]), "\r\n")
		want := "=> " + line
		if got != want {
			t.Errorf("response %d = %q, want %q", i, got, want)
		}
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

// TestWifiStateString verifies the string representation of WifiState.
func TestWifiStateString(t *testing.T) {
	cases := []struct {
		state picnet.WifiState
		want  string
	}{
		{picnet.WifiStateIdle, "idle"},
		{picnet.WifiStateConnecting, "connecting"},
		{picnet.WifiStateConnected, "connected"},
		{picnet.WifiStateError, "error"},
	}
	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("WifiState(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}
