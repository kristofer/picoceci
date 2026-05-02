//go:build tinygo

package freertos

import (
	"container/list"
	"sync"
	"time"
)

// tinygoQueue implements Queue for FreeRTOS.
// This is a placeholder using the same implementation as the stub.
// TODO: Replace with actual FreeRTOS queue calls.
type tinygoQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	items    *list.List
	capacity int
	deleted  bool
	// handle uintptr  // FreeRTOS QueueHandle_t
}

func newQueue(capacity int) Queue {
	// TODO: Call xQueueCreate(capacity, sizeof(uintptr))
	q := &tinygoQueue{
		items:    list.New(),
		capacity: capacity,
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *tinygoQueue) Send(item interface{}, timeoutMs uint32) bool {
	// TODO: Call xQueueSend with timeout
	return q.sendAt(item, timeoutMs, false)
}

func (q *tinygoQueue) SendToFront(item interface{}, timeoutMs uint32) bool {
	// TODO: Call xQueueSendToFront with timeout
	return q.sendAt(item, timeoutMs, true)
}

func (q *tinygoQueue) sendAt(item interface{}, timeoutMs uint32, front bool) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleted {
		return false
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	isForever := timeoutMs == MaxTimeout

	for q.items.Len() >= q.capacity {
		if q.deleted {
			return false
		}
		if timeoutMs == 0 {
			return false
		}
		if !isForever && time.Now().After(deadline) {
			return false
		}
		q.cond.Wait()
	}

	if q.deleted {
		return false
	}

	if front {
		q.items.PushFront(item)
	} else {
		q.items.PushBack(item)
	}
	q.cond.Broadcast()
	return true
}

func (q *tinygoQueue) Receive(timeoutMs uint32) (interface{}, bool) {
	// TODO: Call xQueueReceive with timeout
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleted {
		return nil, false
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	isForever := timeoutMs == MaxTimeout

	for q.items.Len() == 0 {
		if q.deleted {
			return nil, false
		}
		if timeoutMs == 0 {
			return nil, false
		}
		if !isForever && time.Now().After(deadline) {
			return nil, false
		}
		q.cond.Wait()
	}

	if q.deleted || q.items.Len() == 0 {
		return nil, false
	}

	elem := q.items.Front()
	q.items.Remove(elem)
	q.cond.Broadcast()
	return elem.Value, true
}

func (q *tinygoQueue) Peek(timeoutMs uint32) (interface{}, bool) {
	// TODO: Call xQueuePeek with timeout
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleted {
		return nil, false
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	isForever := timeoutMs == MaxTimeout

	for q.items.Len() == 0 {
		if q.deleted {
			return nil, false
		}
		if timeoutMs == 0 {
			return nil, false
		}
		if !isForever && time.Now().After(deadline) {
			return nil, false
		}
		q.cond.Wait()
	}

	if q.deleted || q.items.Len() == 0 {
		return nil, false
	}

	return q.items.Front().Value, true
}

func (q *tinygoQueue) Count() int {
	// TODO: Call uxQueueMessagesWaiting
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.items.Len()
}

func (q *tinygoQueue) SpacesAvailable() int {
	// TODO: Call uxQueueSpacesAvailable
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.capacity - q.items.Len()
}

func (q *tinygoQueue) Capacity() int {
	return q.capacity
}

func (q *tinygoQueue) Delete() {
	// TODO: Call vQueueDelete
	q.mu.Lock()
	defer q.mu.Unlock()
	q.deleted = true
	q.cond.Broadcast()
}

// FreeRTOS function declarations (to be implemented with //go:linkname):
//
// //go:linkname xQueueCreate xQueueCreate
// func xQueueCreate(uxQueueLength uint32, uxItemSize uint32) uintptr
//
// //go:linkname vQueueDelete vQueueDelete
// func vQueueDelete(xQueue uintptr)
//
// //go:linkname xQueueSend xQueueSend
// func xQueueSend(xQueue uintptr, pvItemToQueue unsafe.Pointer, xTicksToWait uint32) int32
//
// //go:linkname xQueueSendToFront xQueueSendToFront
// func xQueueSendToFront(xQueue uintptr, pvItemToQueue unsafe.Pointer, xTicksToWait uint32) int32
//
// //go:linkname xQueueReceive xQueueReceive
// func xQueueReceive(xQueue uintptr, pvBuffer unsafe.Pointer, xTicksToWait uint32) int32
//
// //go:linkname xQueuePeek xQueuePeek
// func xQueuePeek(xQueue uintptr, pvBuffer unsafe.Pointer, xTicksToWait uint32) int32
//
// //go:linkname uxQueueMessagesWaiting uxQueueMessagesWaiting
// func uxQueueMessagesWaiting(xQueue uintptr) uint32
//
// //go:linkname uxQueueSpacesAvailable uxQueueSpacesAvailable
// func uxQueueSpacesAvailable(xQueue uintptr) uint32
