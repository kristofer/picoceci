// Package memory implements the picoceci object allocator.
//
// Default strategy: reference counting with a simple cycle collector.
// On TinyGo/MCU targets the allocator uses a fixed-size heap arena
// drawn from PSRAM.
//
// Phase 3 deliverable — see IMPLEMENTATION_PLAN.md.
// This package is a stub; implementation is pending.
package memory
