//go:build tinygo

package freertos

import (
	"sync"
	"time"
)

// tinygoTask implements TaskHandle for FreeRTOS.
// This is a placeholder - actual implementation requires FreeRTOS bindings.
type tinygoTask struct {
	name     string
	priority uint32
	handle   uintptr // FreeRTOS TaskHandle_t
	deleted  bool
	mu       sync.RWMutex
}

// spawnTask creates a FreeRTOS task.
// TODO: Implement using xTaskCreate via //go:linkname
func spawnTask(name string, fn TaskFunc, stackSize uint16, priority uint32) (TaskHandle, error) {
	// Placeholder implementation using goroutine
	// Real implementation would call xTaskCreate
	t := &tinygoTask{
		name:     name,
		priority: priority,
	}

	registerTask(name, t)

	go func() {
		defer unregisterTask(name)
		fn()
	}()

	return t, nil
}

func (t *tinygoTask) Suspend() {
	// TODO: Call vTaskSuspend(t.handle)
}

func (t *tinygoTask) Resume() {
	// TODO: Call vTaskResume(t.handle)
}

func (t *tinygoTask) Delete() {
	t.mu.Lock()
	t.deleted = true
	t.mu.Unlock()
	// TODO: Call vTaskDelete(t.handle)
}

func (t *tinygoTask) Name() string {
	return t.name
}

func (t *tinygoTask) Priority() uint32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.priority
}

func (t *tinygoTask) SetPriority(priority uint32) {
	t.mu.Lock()
	t.priority = priority
	t.mu.Unlock()
	// TODO: Call vTaskPrioritySet(t.handle, priority)
}

func (t *tinygoTask) IsSuspended() bool {
	// TODO: Check task state via eTaskGetState
	return false
}

func (t *tinygoTask) IsDeleted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.deleted
}

func (t *tinygoTask) Wait() {
	// TODO: Implement task join
	// FreeRTOS doesn't have direct join, would need notification mechanism
}

// delay sleeps for the specified milliseconds.
// TODO: Use vTaskDelay with tick conversion
func delay(ms uint32) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// yield yields to the FreeRTOS scheduler.
// TODO: Call taskYIELD()
func yield() {
	// No-op for now
}

// getTickCount returns the FreeRTOS tick count.
// TODO: Call xTaskGetTickCount()
func getTickCount() uint32 {
	return uint32(time.Since(bootTime).Milliseconds())
}

// FreeRTOS function declarations (to be implemented with //go:linkname):
//
// //go:linkname xTaskCreate xTaskCreate
// func xTaskCreate(pvTaskCode uintptr, pcName *byte, usStackDepth uint16,
//     pvParameters unsafe.Pointer, uxPriority uint32, pxCreatedTask *uintptr) int32
//
// //go:linkname vTaskDelete vTaskDelete
// func vTaskDelete(xTaskToDelete uintptr)
//
// //go:linkname vTaskSuspend vTaskSuspend
// func vTaskSuspend(xTaskToSuspend uintptr)
//
// //go:linkname vTaskResume vTaskResume
// func vTaskResume(xTaskToResume uintptr)
//
// //go:linkname vTaskDelay vTaskDelay
// func vTaskDelay(xTicksToDelay uint32)
//
// //go:linkname taskYIELD taskYIELD
// func taskYIELD()
//
// //go:linkname xTaskGetTickCount xTaskGetTickCount
// func xTaskGetTickCount() uint32
