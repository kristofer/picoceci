//go:build !tinygo

package freertos

import (
	"runtime"
	"sync"
	"time"
)

// stubTask implements TaskHandle using goroutines.
type stubTask struct {
	name      string
	priority  uint32
	fn        TaskFunc
	suspended bool
	deleted   bool
	done      chan struct{}
	suspend   chan struct{}
	resume    chan struct{}
	mu        sync.RWMutex
}

// spawnTask creates a goroutine-based task.
func spawnTask(name string, fn TaskFunc, stackSize uint16, priority uint32) (TaskHandle, error) {
	t := &stubTask{
		name:     name,
		priority: priority,
		fn:       fn,
		done:     make(chan struct{}),
		suspend:  make(chan struct{}, 1),
		resume:   make(chan struct{}, 1),
	}

	registerTask(name, t)

	go func() {
		defer func() {
			close(t.done)
			unregisterTask(name)
		}()

		// Check for suspension before running
		t.checkSuspend()

		// Run the task function
		if !t.IsDeleted() {
			fn()
		}
	}()

	return t, nil
}

// checkSuspend blocks if the task is suspended.
func (t *stubTask) checkSuspend() {
	for {
		t.mu.RLock()
		if !t.suspended || t.deleted {
			t.mu.RUnlock()
			return
		}
		t.mu.RUnlock()

		// Wait for resume signal
		select {
		case <-t.resume:
			return
		case <-time.After(10 * time.Millisecond):
			// Check again
		}
	}
}

func (t *stubTask) Suspend() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.suspended = true
}

func (t *stubTask) Resume() {
	t.mu.Lock()
	t.suspended = false
	t.mu.Unlock()

	// Signal resume
	select {
	case t.resume <- struct{}{}:
	default:
	}
}

func (t *stubTask) Delete() {
	t.mu.Lock()
	t.deleted = true
	t.mu.Unlock()

	// Resume if suspended so it can exit
	t.Resume()
}

func (t *stubTask) Name() string {
	return t.name
}

func (t *stubTask) Priority() uint32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.priority
}

func (t *stubTask) SetPriority(priority uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.priority = priority
}

func (t *stubTask) IsSuspended() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.suspended
}

func (t *stubTask) IsDeleted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.deleted
}

func (t *stubTask) Wait() {
	<-t.done
}

// delay sleeps for the specified milliseconds.
func delay(ms uint32) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// yield yields to the Go scheduler.
func yield() {
	runtime.Gosched()
}

// getTickCount returns milliseconds since program start.
func getTickCount() uint32 {
	return uint32(time.Since(bootTime).Milliseconds())
}
