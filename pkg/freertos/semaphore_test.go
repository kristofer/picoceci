package freertos_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kristofer/picoceci/pkg/freertos"
)

func TestBinarySemaphore(t *testing.T) {
	sem := freertos.NewBinarySemaphore()
	defer sem.Delete()

	// Binary semaphore starts "taken" - Take should fail
	if sem.Take(0) {
		t.Error("Take on new binary semaphore should fail")
	}

	// Give should succeed
	if !sem.Give() {
		t.Error("Give failed")
	}

	// Now Take should succeed
	if !sem.Take(0) {
		t.Error("Take after Give failed")
	}

	// Second Take should fail (binary)
	if sem.Take(0) {
		t.Error("Second Take should fail on binary semaphore")
	}
}

func TestCountingSemaphore(t *testing.T) {
	sem := freertos.NewCountingSemaphore(3, 0)
	defer sem.Delete()

	// Give 3 times
	for i := 0; i < 3; i++ {
		if !sem.Give() {
			t.Errorf("Give %d failed", i)
		}
	}

	// 4th Give should fail (at max)
	if sem.Give() {
		t.Error("Give beyond max should fail")
	}

	// Take 3 times
	for i := 0; i < 3; i++ {
		if !sem.Take(0) {
			t.Errorf("Take %d failed", i)
		}
	}

	// 4th Take should fail
	if sem.Take(0) {
		t.Error("Take beyond count should fail")
	}
}

func TestCountingSemaphoreInitialCount(t *testing.T) {
	sem := freertos.NewCountingSemaphore(5, 3)
	defer sem.Delete()

	// Should be able to Take 3 times
	for i := 0; i < 3; i++ {
		if !sem.Take(0) {
			t.Errorf("Take %d failed with initial count 3", i)
		}
	}

	// 4th Take should fail
	if sem.Take(0) {
		t.Error("Take beyond initial count should fail")
	}
}

func TestMutex(t *testing.T) {
	mutex := freertos.NewMutex()
	defer mutex.Delete()

	// Mutex starts unlocked
	if !mutex.Take(0) {
		t.Error("Initial Take on mutex failed")
	}

	// Second Take should fail (mutex is held)
	if mutex.Take(0) {
		t.Error("Second Take on mutex should fail")
	}

	// Give to unlock
	if !mutex.Give() {
		t.Error("Give failed")
	}

	// Take should succeed again
	if !mutex.Take(0) {
		t.Error("Take after Give failed")
	}
}

func TestSemaphoreTimeout(t *testing.T) {
	sem := freertos.NewBinarySemaphore()
	defer sem.Delete()

	start := time.Now()
	ok := sem.Take(50)
	elapsed := time.Since(start)

	if ok {
		t.Error("Take should have timed out")
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("Take returned too quickly: %v", elapsed)
	}
}

func TestSemaphoreGetCount(t *testing.T) {
	sem := freertos.NewCountingSemaphore(10, 0)
	defer sem.Delete()

	if sem.GetCount() != 0 {
		t.Errorf("initial count = %d, want 0", sem.GetCount())
	}

	sem.Give()
	sem.Give()
	sem.Give()

	if sem.GetCount() != 3 {
		t.Errorf("count after 3 Give = %d, want 3", sem.GetCount())
	}

	sem.Take(0)

	if sem.GetCount() != 2 {
		t.Errorf("count after Take = %d, want 2", sem.GetCount())
	}
}

func TestMutexProtection(t *testing.T) {
	mutex := freertos.NewMutex()
	defer mutex.Delete()

	var counter int32
	var wg sync.WaitGroup

	// Spawn multiple tasks that increment counter
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mutex.Take(freertos.MaxTimeout)
				// Critical section
				current := atomic.LoadInt32(&counter)
				time.Sleep(time.Microsecond)
				atomic.StoreInt32(&counter, current+1)
				mutex.Give()
			}
		}()
	}

	wg.Wait()

	// Without mutex protection, counter would be less than 1000 due to races
	if counter != 1000 {
		t.Errorf("counter = %d, want 1000", counter)
	}
}

func TestSemaphoreDelete(t *testing.T) {
	sem := freertos.NewBinarySemaphore()
	sem.Give()

	sem.Delete()

	// Operations on deleted semaphore should fail
	if sem.Take(0) {
		t.Error("Take on deleted semaphore succeeded")
	}
	if sem.Give() {
		t.Error("Give on deleted semaphore succeeded")
	}
}
