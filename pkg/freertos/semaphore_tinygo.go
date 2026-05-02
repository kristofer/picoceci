//go:build tinygo

package freertos

import (
	"sync"
	"time"
)

// tinygoSemaphore implements Semaphore for FreeRTOS.
// This placeholder uses a channel-based implementation.
// TODO: Replace with actual FreeRTOS semaphore calls.
type tinygoSemaphore struct {
	kind     SemaphoreKind
	maxCount int
	ch       chan struct{}
	mu       sync.Mutex
	deleted  bool
	// handle uintptr  // FreeRTOS SemaphoreHandle_t
}

func newSemaphore(kind SemaphoreKind, maxCount int) Semaphore {
	// TODO: Call xSemaphoreCreateBinary/xSemaphoreCreateCounting/xSemaphoreCreateMutex
	return &tinygoSemaphore{
		kind:     kind,
		maxCount: maxCount,
		ch:       make(chan struct{}, maxCount),
	}
}

func (s *tinygoSemaphore) Take(timeoutMs uint32) bool {
	// TODO: Call xSemaphoreTake
	s.mu.Lock()
	if s.deleted {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	if timeoutMs == 0 {
		select {
		case <-s.ch:
			return true
		default:
			return false
		}
	}

	if timeoutMs == MaxTimeout {
		select {
		case <-s.ch:
			return true
		}
	}

	select {
	case <-s.ch:
		return true
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return false
	}
}

func (s *tinygoSemaphore) Give() bool {
	// TODO: Call xSemaphoreGive
	s.mu.Lock()
	if s.deleted {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *tinygoSemaphore) GetCount() int {
	return len(s.ch)
}

func (s *tinygoSemaphore) Delete() {
	// TODO: Call vSemaphoreDelete
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = true
	close(s.ch)
}

// FreeRTOS function declarations (to be implemented with //go:linkname):
//
// //go:linkname xSemaphoreCreateBinary xSemaphoreCreateBinary
// func xSemaphoreCreateBinary() uintptr
//
// //go:linkname xSemaphoreCreateCounting xSemaphoreCreateCounting
// func xSemaphoreCreateCounting(uxMaxCount uint32, uxInitialCount uint32) uintptr
//
// //go:linkname xSemaphoreCreateMutex xSemaphoreCreateMutex
// func xSemaphoreCreateMutex() uintptr
//
// //go:linkname vSemaphoreDelete vSemaphoreDelete
// func vSemaphoreDelete(xSemaphore uintptr)
//
// //go:linkname xSemaphoreTake xSemaphoreTake
// func xSemaphoreTake(xSemaphore uintptr, xTicksToWait uint32) int32
//
// //go:linkname xSemaphoreGive xSemaphoreGive
// func xSemaphoreGive(xSemaphore uintptr) int32
