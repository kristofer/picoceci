//go:build !tinygo

package freertos

import (
	"sync"
	"time"
)

// stubQueue implements Queue using a channel.
type stubQueue struct {
	ch       chan interface{}
	capacity int
	mu       sync.Mutex
	deleted  bool
}

func newQueue(capacity int) Queue {
	return &stubQueue{
		ch:       make(chan interface{}, capacity),
		capacity: capacity,
	}
}

func (q *stubQueue) Send(item interface{}, timeoutMs uint32) bool {
	q.mu.Lock()
	if q.deleted {
		q.mu.Unlock()
		return false
	}
	q.mu.Unlock()

	if timeoutMs == 0 {
		select {
		case q.ch <- item:
			return true
		default:
			return false
		}
	}

	if timeoutMs == MaxTimeout {
		select {
		case q.ch <- item:
			return true
		}
	}

	select {
	case q.ch <- item:
		return true
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return false
	}
}

func (q *stubQueue) SendToFront(item interface{}, timeoutMs uint32) bool {
	// Go channels don't support front insertion
	// For the stub, we'll just send normally
	// A real FreeRTOS implementation would use xQueueSendToFront
	return q.Send(item, timeoutMs)
}

func (q *stubQueue) Receive(timeoutMs uint32) (interface{}, bool) {
	q.mu.Lock()
	if q.deleted {
		q.mu.Unlock()
		return nil, false
	}
	q.mu.Unlock()

	if timeoutMs == 0 {
		select {
		case item := <-q.ch:
			return item, true
		default:
			return nil, false
		}
	}

	if timeoutMs == MaxTimeout {
		select {
		case item := <-q.ch:
			return item, true
		}
	}

	select {
	case item := <-q.ch:
		return item, true
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return nil, false
	}
}

func (q *stubQueue) Peek(timeoutMs uint32) (interface{}, bool) {
	// Go channels don't support peek
	// For the stub, this is a limitation
	// We'll receive and immediately send back (may change order)
	item, ok := q.Receive(timeoutMs)
	if ok {
		// Try to put it back - this is imperfect but works for testing
		select {
		case q.ch <- item:
		default:
			// Queue is full, item is lost - this is a stub limitation
		}
	}
	return item, ok
}

func (q *stubQueue) Count() int {
	return len(q.ch)
}

func (q *stubQueue) SpacesAvailable() int {
	return q.capacity - len(q.ch)
}

func (q *stubQueue) Capacity() int {
	return q.capacity
}

func (q *stubQueue) Delete() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.deleted = true
	close(q.ch)
}
