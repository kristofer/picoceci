//go:build tinygo

package net

import (
	"fmt"
)

// tinygoManagerImpl is a placeholder for the TinyGo/ESP32 WiFi backend.
// Full implementation requires tinygo/net and machine/wifi packages.
type tinygoManagerImpl struct {
	state WifiState
	ip    string
}

func newManagerImpl() managerImpl {
	return &tinygoManagerImpl{state: WifiStateIdle}
}

func (t *tinygoManagerImpl) connect(ssid, password string) error {
	// TODO: implement using tinygo/net or machine/wifi.
	// For now, mark as error so failures surface as IOError.
	t.state = WifiStateError
	return fmt.Errorf("WiFi not yet implemented on this TinyGo target")
}

func (t *tinygoManagerImpl) status() WifiState {
	return t.state
}

func (t *tinygoManagerImpl) ipAddress() string {
	return t.ip
}

func (t *tinygoManagerImpl) listen(port int) (Listener, error) {
	return nil, fmt.Errorf("WiFi not yet implemented on this TinyGo target")
}

func (t *tinygoManagerImpl) disconnect() error {
	t.state = WifiStateIdle
	t.ip = ""
	return nil
}
