// Package freertos exposes FreeRTOS concurrency primitives as picoceci objects.
//
// See docs/freertos-bridge.md for the full design and FreeRTOS API mapping.
//
// Build tags:
//   - tinygo  — real FreeRTOS calls via TinyGo's unsafe/linkname mechanism.
//   - !tinygo — goroutine/channel-based stubs for desktop testing.
//
// Phase 5 deliverable — see IMPLEMENTATION_PLAN.md.
// This package is a stub; implementation is pending.
package freertos
