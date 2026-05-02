package freertos_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/kristofer/picoceci/pkg/freertos"
)

func TestTaskSpawnAndComplete(t *testing.T) {
	var executed int32

	task, err := freertos.SpawnTask("test-spawn", func() {
		atomic.StoreInt32(&executed, 1)
	}, 1024, 1)
	if err != nil {
		t.Fatalf("SpawnTask error: %v", err)
	}

	task.Wait()

	if atomic.LoadInt32(&executed) != 1 {
		t.Error("task function was not executed")
	}
}

func TestTaskName(t *testing.T) {
	task, _ := freertos.SpawnTask("named-task", func() {}, 1024, 1)
	defer task.Wait()

	if task.Name() != "named-task" {
		t.Errorf("Name() = %q, want %q", task.Name(), "named-task")
	}
}

func TestTaskPriority(t *testing.T) {
	task, _ := freertos.SpawnTask("priority-task", func() {
		freertos.Delay(50)
	}, 1024, 5)
	defer task.Wait()

	if task.Priority() != 5 {
		t.Errorf("Priority() = %d, want 5", task.Priority())
	}

	task.SetPriority(10)
	if task.Priority() != 10 {
		t.Errorf("after SetPriority, Priority() = %d, want 10", task.Priority())
	}
}

func TestTaskSuspendResume(t *testing.T) {
	var step int32

	task, _ := freertos.SpawnTask("suspend-task", func() {
		atomic.StoreInt32(&step, 1)
		// Task will be suspended here by the test
		for i := 0; i < 100; i++ {
			freertos.Delay(10)
			if atomic.LoadInt32(&step) == 2 {
				break
			}
		}
		atomic.StoreInt32(&step, 3)
	}, 1024, 1)

	// Wait for task to start
	time.Sleep(20 * time.Millisecond)

	// Verify task started
	if atomic.LoadInt32(&step) != 1 {
		t.Error("task did not start")
	}

	// Suspend and verify
	task.Suspend()
	if !task.IsSuspended() {
		t.Error("IsSuspended() = false after Suspend")
	}

	// Signal task to proceed
	atomic.StoreInt32(&step, 2)

	// Resume
	task.Resume()
	if task.IsSuspended() {
		t.Error("IsSuspended() = true after Resume")
	}

	task.Wait()

	if atomic.LoadInt32(&step) != 3 {
		t.Error("task did not complete after resume")
	}
}

func TestTaskDelete(t *testing.T) {
	running := make(chan struct{})
	deleted := make(chan struct{})

	task, _ := freertos.SpawnTask("delete-task", func() {
		close(running)
		// Wait to be deleted
		<-deleted
	}, 1024, 1)

	// Wait for task to start
	<-running

	task.Delete()

	if !task.IsDeleted() {
		t.Error("IsDeleted() = false after Delete")
	}

	close(deleted)
}

func TestDelay(t *testing.T) {
	start := time.Now()
	freertos.Delay(50)
	elapsed := time.Since(start)

	// Allow some tolerance
	if elapsed < 45*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("Delay(50) took %v, expected ~50ms", elapsed)
	}
}

func TestYield(t *testing.T) {
	// Just verify it doesn't panic
	freertos.Yield()
}

func TestGetTickCount(t *testing.T) {
	tick1 := freertos.GetTickCount()
	freertos.Delay(10)
	tick2 := freertos.GetTickCount()

	if tick2 <= tick1 {
		t.Errorf("GetTickCount did not advance: %d -> %d", tick1, tick2)
	}
}

func TestGetTask(t *testing.T) {
	done := make(chan struct{})

	task, _ := freertos.SpawnTask("lookup-task", func() {
		<-done
	}, 1024, 1)

	// Should be able to look up by name
	found := freertos.GetTask("lookup-task")
	if found == nil {
		t.Error("GetTask returned nil for existing task")
	}
	if found != task {
		t.Error("GetTask returned different handle")
	}

	// Non-existent task
	notFound := freertos.GetTask("no-such-task")
	if notFound != nil {
		t.Error("GetTask returned non-nil for non-existent task")
	}

	close(done)
	task.Wait()
}

func TestListTasks(t *testing.T) {
	done1 := make(chan struct{})
	done2 := make(chan struct{})

	task1, _ := freertos.SpawnTask("list-task-1", func() { <-done1 }, 1024, 1)
	task2, _ := freertos.SpawnTask("list-task-2", func() { <-done2 }, 1024, 1)

	names := freertos.ListTasks()

	found1, found2 := false, false
	for _, name := range names {
		if name == "list-task-1" {
			found1 = true
		}
		if name == "list-task-2" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("ListTasks missing tasks: %v", names)
	}

	close(done1)
	close(done2)
	task1.Wait()
	task2.Wait()
}

func TestMultipleTasks(t *testing.T) {
	const numTasks = 5
	var counter int32

	tasks := make([]freertos.TaskHandle, numTasks)
	for i := 0; i < numTasks; i++ {
		var err error
		tasks[i], err = freertos.SpawnTask("multi-task", func() {
			atomic.AddInt32(&counter, 1)
		}, 1024, 1)
		if err != nil {
			t.Fatalf("SpawnTask %d error: %v", i, err)
		}
	}

	// Wait for all tasks
	for _, task := range tasks {
		task.Wait()
	}

	if atomic.LoadInt32(&counter) != numTasks {
		t.Errorf("counter = %d, want %d", counter, numTasks)
	}
}
