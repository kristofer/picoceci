# FreeRTOS Bridge — picoceci ↔ TinyGo Runtime

Version: 0.1-draft  
Audience: Implementors of `pkg/freertos/` and `pkg/tinygo/`

---

## Overview

picoceci exposes FreeRTOS concurrency primitives as first-class objects.  This document describes how each picoceci object maps to a FreeRTOS C API call, the TinyGo declaration required to import that call, and the runtime glue that connects the two.

The bridge layer lives in `pkg/freertos/` and uses build constraints to swap between:

- **TinyGo target** (`//go:build tinygo`) — real FreeRTOS calls via `unsafe`/`//export`
- **Desktop stub** (`//go:build !tinygo`) — goroutine + channel based equivalents for testing

---

## Build constraint pattern

```go
// pkg/freertos/task_tinygo.go
//go:build tinygo

package freertos

import "unsafe"

//go:linkname xTaskCreate xTaskCreate
func xTaskCreate(
    pvTaskCode    uintptr,
    pcName        *byte,
    usStackDepth  uint16,
    pvParameters  unsafe.Pointer,
    uxPriority    uint32,
    pxCreatedTask *uintptr,
) int32
```

```go
// pkg/freertos/task_stub.go
//go:build !tinygo

package freertos

import "sync"

// Desktop stub: FreeRTOS tasks become goroutines.

func xTaskCreate(fn uintptr, name *byte, stackDepth uint16,
    params unsafe.Pointer, priority uint32, handle *uintptr) int32 {
    // ... goroutine stub
    return 1 // pdPASS
}
```

---

## Task API

### FreeRTOS functions used

| FreeRTOS C API | Purpose |
|---|---|
| `xTaskCreate` | Create and start a task |
| `vTaskDelete` | Delete a task |
| `vTaskSuspend` | Suspend a task |
| `vTaskResume` | Resume a suspended task |
| `vTaskDelay` | Delay for a number of ticks |
| `vTaskDelayUntil` | Delay until an absolute tick count |
| `uxTaskPriorityGet` | Get task priority |
| `vTaskPrioritySet` | Set task priority |
| `xTaskGetCurrentTaskHandle` | Get handle of current task |
| `pcTaskGetName` | Get task name |
| `taskYIELD` | Yield to scheduler |

### Tick conversion

All picoceci time values are in **milliseconds**.  Convert to FreeRTOS ticks:

```go
const pdMS_TO_TICKS = 1  // when configTICK_RATE_HZ == 1000
// tick = ms * configTICK_RATE_HZ / 1000
```

### picoceci `Task` object

```
Task {
    handle   uintptr    // FreeRTOS TaskHandle_t
    block    *Object    // the picoceci block to run
    name     string
    priority uint32
    stackSz  uint16
}
```

#### `Task spawn: aBlock`

1. Allocate a `Task` object.
2. Serialize the picoceci block reference into a pointer-sized token.
3. Call `xTaskCreate` with a trampoline function `taskTrampoline(pvParams unsafe.Pointer)`.
4. `taskTrampoline` re-hydrates the block reference and calls `block value`.

```go
func taskTrampoline(params unsafe.Pointer) {
    task := (*picoTask)(params)
    vm := task.vm
    _, _ = vm.CallBlock(task.block, nil)
    // task ends when block returns
    xTaskDelete(0) // delete self
}
```

#### `task suspend` / `task resume`

Direct calls to `vTaskSuspend(handle)` / `vTaskResume(handle)`.

#### `Task delay: ms`

```go
vTaskDelay(pdMS_TO_TICKS(ms))
```

#### `Task yield`

```go
taskYIELD()
```

---

## Queue API

### FreeRTOS functions used

| FreeRTOS C API | Purpose |
|---|---|
| `xQueueCreate` | Create queue |
| `vQueueDelete` | Delete queue |
| `xQueueSend` | Enqueue item (back) |
| `xQueueSendToFront` | Enqueue item (front) |
| `xQueueReceive` | Dequeue item |
| `xQueuePeek` | Peek without removing |
| `uxQueueMessagesWaiting` | Number of items waiting |
| `uxQueueSpacesAvailable` | Free slots |
| `xQueueSendFromISR` | Send from interrupt (used by Timer bridge) |

### Item representation

FreeRTOS queues are typed; picoceci queues transfer pointer-sized tokens (a `uintptr` holding an `*Object` pointer with ref-count retained).  Item size is therefore `sizeof(uintptr)` = 4 bytes on 32-bit targets.

```go
func queueSend(q uintptr, obj *Object, timeoutTicks uint32) bool {
    Retain(obj)
    token := uintptr(unsafe.Pointer(obj))
    result := xQueueSend(q, unsafe.Pointer(&token), timeoutTicks)
    if result == 0 { // pdFAIL
        Release(obj)
        return false
    }
    return true
}

func queueReceive(q uintptr, timeoutTicks uint32) (*Object, bool) {
    var token uintptr
    result := xQueueReceive(q, unsafe.Pointer(&token), timeoutTicks)
    if result == 0 {
        return nil, false
    }
    obj := (*Object)(unsafe.Pointer(token))
    // caller owns one reference
    return obj, true
}
```

### Timeout sentinel

`portMAX_DELAY` (0xFFFFFFFF) is used for blocking-forever semantics.

---

## Semaphore API

### FreeRTOS functions used

| FreeRTOS C API | Purpose |
|---|---|
| `xSemaphoreCreateBinary` | Binary semaphore |
| `xSemaphoreCreateCounting` | Counting semaphore |
| `xSemaphoreCreateMutex` | Recursive mutex |
| `vSemaphoreDelete` | Delete semaphore |
| `xSemaphoreTake` | Acquire (wait) |
| `xSemaphoreGive` | Release |
| `xSemaphoreGiveFromISR` | Release from ISR |

### picoceci `Semaphore` object

```
Semaphore {
    handle uintptr   // SemaphoreHandle_t
    kind   uint8     // 0=binary 1=counting 2=mutex
}
```

#### `Semaphore new`

`xSemaphoreCreateBinary()` — starts in "taken" state; caller must `give` before it can be `take`n.

#### `Semaphore counting: maxCount`

`xSemaphoreCreateCounting(maxCount, 0)` — starts at count 0.

#### `Semaphore mutex`

`xSemaphoreCreateMutex()` — recursive mutex with priority inheritance.

#### `sem take` / `sem take timeout: ms`

```go
xSemaphoreTake(handle, ticks)
```

Returns `true` on success, `false` on timeout.  Blocking variant uses `portMAX_DELAY`.

#### `sem give`

```go
xSemaphoreGive(handle)
```

---

## Timer API

### FreeRTOS functions used

| FreeRTOS C API | Purpose |
|---|---|
| `xTimerCreate` | Create software timer |
| `xTimerDelete` | Delete timer |
| `xTimerStart` | Start timer |
| `xTimerStop` | Stop timer |
| `xTimerReset` | Reset timer period |
| `xTimerChangePeriod` | Change period |
| `pvTimerGetTimerID` | Get user data |

### One-shot vs periodic

```
Timer {
    handle   uintptr   // TimerHandle_t
    block    *Object   // callback block
    periodic bool
}
```

`Timer after: ms do: aBlock` → `xTimerCreate(..., ms * pdMS_PER_TICK, pdFALSE, ...)`  
`Timer every: ms do: aBlock` → `xTimerCreate(..., ms * pdMS_PER_TICK, pdTRUE, ...)`

### Timer callback trampoline

```go
func timerCallback(xTimer uintptr) {
    id := pvTimerGetTimerID(xTimer)
    timer := (*picoTimer)(unsafe.Pointer(id))
    vm := timer.vm
    _, _ = vm.CallBlock(timer.block, nil)
}
```

The callback runs in the FreeRTOS timer daemon task.  The picoceci VM must be re-entrant or protected by a mutex.

---

## Interrupt Safety

Certain FreeRTOS functions have `FromISR` variants that must be called from interrupt service routines.  The picoceci GPIO interrupt bridge (`gpio onEdge: #rising do: aBlock`) posts a message to an internal queue from the ISR and a dedicated task dequeues and evaluates the block — ensuring the block runs in task context, not interrupt context.

---

## Desktop Stubs

All FreeRTOS functionality is stubbed using Go's native `sync` and `time` packages when built without the `tinygo` constraint.  This allows:

- Unit testing the interpreter without hardware
- Running integration tests on CI runners
- Debugging with Go's race detector

### Mapping

| FreeRTOS | Desktop stub |
|---|---|
| `xTaskCreate` | `go func()` goroutine |
| `vTaskDelay` | `time.Sleep` |
| `xQueueCreate/Send/Receive` | buffered `chan interface{}` |
| `xSemaphoreCreate/Take/Give` | `sync.Mutex` / `sync.Cond` |
| `xTimerCreate/Start` | `time.AfterFunc` |

---

## Error Handling

| FreeRTOS return | picoceci effect |
|---|---|
| `pdPASS` / `pdTRUE` | Success; return value as per method doc |
| `pdFAIL` / `pdFALSE` | Return `false` (timeout-based methods) |
| Resource creation failure (NULL) | Raise `TaskError signal: 'FreeRTOS resource allocation failed'` |

---

## Memory Notes

Each FreeRTOS object (task stack, queue buffer, semaphore block) is allocated from FreeRTOS heap (`pvPortMalloc`).  The picoceci GC/RC is independent of the FreeRTOS heap.  When a picoceci `Task`, `Queue`, `Semaphore`, or `Timer` object is garbage-collected, a finalizer calls the corresponding `vDelete`/`vSemaphoreDelete` function to return memory to FreeRTOS.

On ESP32-S3, configure FreeRTOS to use PSRAM for its heap to preserve internal SRAM for ISR stacks:

```c
// sdkconfig or TinyGo target configuration
CONFIG_FREERTOS_USE_TRACE_FACILITY=y
CONFIG_SPIRAM_USE_MALLOC=y
CONFIG_SPIRAM_ALLOW_STACK_EXTERNAL_MEMORY=y
```

---

*End of freertos-bridge.md*
