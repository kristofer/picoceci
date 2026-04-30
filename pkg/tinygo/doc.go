// Package tinygo provides TinyGo-specific runtime shims for picoceci.
//
// When building for a desktop host (go:build !tinygo), this package
// provides stub implementations that map to standard Go libraries so
// that the full test suite can run without hardware.
//
// When building with TinyGo (go:build tinygo), this package maps to
// TinyGo's machine package and FreeRTOS bindings.
//
// Phase 5 deliverable — see IMPLEMENTATION_PLAN.md.
// This package is a stub; implementation is pending.
package tinygo
