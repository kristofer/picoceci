//go:build !tinygo

package freertos

import (
	"sync"
	"time"
)

// stubSemaphore implements Semaphore using a channel.
type stubSemaphore struct {
	kind     SemaphoreKind
	maxCount int
	ch       chan struct{}
	mu       sync.Mutex
	deleted  bool
}

func newSemaphore(kind SemaphoreKind, maxCount int) Semaphore {
	return &stubSemaphore{
		kind:     kind,
		maxCount: maxCount,
		ch:       make(chan struct{}, maxCount),
	}
}

func (s *stubSemaphore) Take(timeoutMs uint32) bool {
	s.mu.Lock()
	if s.deleted {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	if timeoutMs == 0 {
		// Non-blocking
		select {
		case <-s.ch:
			return true
		default:
			return false
		}
	}

	if timeoutMs == MaxTimeout {
		// Blocking forever
		select {
		case <-s.ch:
			return true
		}
	}

	// Blocking with timeout
	select {
	case <-s.ch:
		return true
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return false
	}
}

func (s *stubSemaphore) Give() bool {
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
		return false // Channel full (at max count)
	}
}

func (s *stubSemaphore) GetCount() int {
	return len(s.ch)
}

func (s *stubSemaphore) Delete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = true
	// Drain the channel to unblock any waiting Takes
	close(s.ch)
}
