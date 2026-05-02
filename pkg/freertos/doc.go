// Package freertos exposes FreeRTOS concurrency primitives as picoceci objects.
//
// See docs/freertos-bridge.md for the full design and FreeRTOS API mapping.
//
// # Task
//
// Tasks are lightweight concurrent execution units:
//
//	task, _ := freertos.SpawnTask("worker", func() {
//	    for i := 0; i < 10; i++ {
//	        freertos.Delay(100)
//	    }
//	}, 1024, 1)
//	task.Wait()
//
// # Queue
//
// Queues enable thread-safe communication between tasks:
//
//	q := freertos.NewQueue(10)
//	q.Send("message", freertos.MaxTimeout)
//	item, ok := q.Receive(freertos.MaxTimeout)
//
// # Semaphore
//
// Semaphores provide synchronization:
//
//	mutex := freertos.NewMutex()
//	mutex.Take(freertos.MaxTimeout)
//	// critical section
//	mutex.Give()
//
// # Timer
//
// Timers schedule callback execution:
//
//	timer := freertos.TimerEvery(1000, func() {
//	    println("tick")
//	})
//	timer.Stop()
//
// # Build Tags
//
//   - tinygo  — real FreeRTOS calls via TinyGo's unsafe/linkname mechanism
//   - !tinygo — goroutine/channel-based stubs for desktop testing
//
// Phase 5 deliverable — see IMPLEMENTATION_PLAN.md.
package freertos
