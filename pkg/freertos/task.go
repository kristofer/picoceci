package freertos

import (
	"sync"
	"time"
)

// TaskFunc is a function that can be run as a task.
// This is used instead of picoceci blocks for the core implementation.
// The picoceci object wrapper will convert blocks to TaskFuncs.
type TaskFunc func()

// TaskHandle represents a running task.
type TaskHandle interface {
	// Suspend pauses the task.
	Suspend()

	// Resume resumes a suspended task.
	Resume()

	// Delete terminates the task.
	Delete()

	// Name returns the task name.
	Name() string

	// Priority returns the task priority (0-255).
	Priority() uint32

	// SetPriority changes the task priority.
	SetPriority(priority uint32)

	// IsSuspended returns true if the task is suspended.
	IsSuspended() bool

	// IsDeleted returns true if the task has been deleted.
	IsDeleted() bool

	// Wait blocks until the task completes or is deleted.
	Wait()
}

// SpawnTask creates and starts a new task.
// name: task identifier
// fn: the function to run
// stackSize: stack size in bytes (ignored on desktop)
// priority: task priority 0-255 (higher = more priority)
func SpawnTask(name string, fn TaskFunc, stackSize uint16, priority uint32) (TaskHandle, error) {
	return spawnTask(name, fn, stackSize, priority)
}

// Delay suspends the current task for the specified duration.
// This maps to vTaskDelay on FreeRTOS.
func Delay(ms uint32) {
	delay(ms)
}

// Yield yields execution to other tasks.
// This maps to taskYIELD on FreeRTOS.
func Yield() {
	yield()
}

// GetTickCount returns the current tick count since boot.
// On desktop, this is milliseconds since program start.
func GetTickCount() uint32 {
	return getTickCount()
}

// taskRegistry tracks all tasks for management and testing.
var taskRegistry = struct {
	sync.RWMutex
	tasks map[string]TaskHandle
}{
	tasks: make(map[string]TaskHandle),
}

// registerTask adds a task to the registry.
func registerTask(name string, handle TaskHandle) {
	taskRegistry.Lock()
	defer taskRegistry.Unlock()
	taskRegistry.tasks[name] = handle
}

// unregisterTask removes a task from the registry.
func unregisterTask(name string) {
	taskRegistry.Lock()
	defer taskRegistry.Unlock()
	delete(taskRegistry.tasks, name)
}

// GetTask returns a task by name, or nil if not found.
func GetTask(name string) TaskHandle {
	taskRegistry.RLock()
	defer taskRegistry.RUnlock()
	return taskRegistry.tasks[name]
}

// ListTasks returns all registered task names.
func ListTasks() []string {
	taskRegistry.RLock()
	defer taskRegistry.RUnlock()
	names := make([]string, 0, len(taskRegistry.tasks))
	for name := range taskRegistry.tasks {
		names = append(names, name)
	}
	return names
}

// bootTime is used for tick count calculation.
var bootTime = time.Now()
