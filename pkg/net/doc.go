// Package net provides WiFi and TCP session management for picoceci.
//
// On TinyGo/ESP32-S3 targets, this package wraps the WiFi chip and opens
// a TCP listener that accepts incoming REPL sessions.  On desktop (non-TinyGo)
// targets it provides a standard net.Listen-backed stub suitable for testing.
//
// Typical usage pattern (picoceci programs):
//
//	Wifi connectSSID: 'mynet' password: 'secret'.
//	Wifi listenOn: 2323 do: [ :session |
//	    Task spawn: [ PicoceciREPL serve: session ] name: 'tcp-session'.
//	].
//
// Phase 6 deliverable — see docs/IMPLEMENTATION_PLAN.md.
package net
