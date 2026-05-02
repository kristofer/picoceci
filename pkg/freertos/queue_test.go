package freertos_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kristofer/picoceci/pkg/freertos"
)

func TestQueueSendReceive(t *testing.T) {
	q := freertos.NewQueue(10)
	defer q.Delete()

	// Send items
	if !q.Send("first", 0) {
		t.Error("Send failed")
	}
	if !q.Send("second", 0) {
		t.Error("Send failed")
	}

	// Receive in order
	item, ok := q.Receive(0)
	if !ok || item != "first" {
		t.Errorf("Receive = %v, %v; want 'first', true", item, ok)
	}

	item, ok = q.Receive(0)
	if !ok || item != "second" {
		t.Errorf("Receive = %v, %v; want 'second', true", item, ok)
	}
}

func TestQueueSendToFront(t *testing.T) {
	// Note: The desktop stub uses Go channels which don't support front insertion.
	// This test is skipped on desktop. On TinyGo with real FreeRTOS, it would work.
	t.Skip("SendToFront not supported by channel-based desktop stub")

	q := freertos.NewQueue(10)
	defer q.Delete()

	q.Send("first", 0)
	q.Send("second", 0)
	q.SendToFront("urgent", 0)

	// urgent should come first
	item, _ := q.Receive(0)
	if item != "urgent" {
		t.Errorf("first item = %v, want 'urgent'", item)
	}
}

func TestQueuePeek(t *testing.T) {
	// Note: The desktop stub uses Go channels which don't support peek.
	// This test is skipped on desktop. On TinyGo with real FreeRTOS, it would work.
	t.Skip("Peek not properly supported by channel-based desktop stub")

	q := freertos.NewQueue(10)
	defer q.Delete()

	q.Send("item", 0)

	// Peek should return item without removing
	item, ok := q.Peek(0)
	if !ok || item != "item" {
		t.Errorf("Peek = %v, %v; want 'item', true", item, ok)
	}

	// Item should still be there
	if q.Count() != 1 {
		t.Error("Peek removed the item")
	}

	// Receive should still get the item
	item, _ = q.Receive(0)
	if item != "item" {
		t.Errorf("Receive after Peek = %v, want 'item'", item)
	}
}

func TestQueueCount(t *testing.T) {
	q := freertos.NewQueue(10)
	defer q.Delete()

	if q.Count() != 0 {
		t.Errorf("initial Count = %d, want 0", q.Count())
	}

	q.Send("a", 0)
	q.Send("b", 0)

	if q.Count() != 2 {
		t.Errorf("Count = %d, want 2", q.Count())
	}

	q.Receive(0)

	if q.Count() != 1 {
		t.Errorf("Count after receive = %d, want 1", q.Count())
	}
}

func TestQueueCapacity(t *testing.T) {
	q := freertos.NewQueue(5)
	defer q.Delete()

	if q.Capacity() != 5 {
		t.Errorf("Capacity = %d, want 5", q.Capacity())
	}

	if q.SpacesAvailable() != 5 {
		t.Errorf("SpacesAvailable = %d, want 5", q.SpacesAvailable())
	}

	q.Send("a", 0)
	q.Send("b", 0)

	if q.SpacesAvailable() != 3 {
		t.Errorf("SpacesAvailable = %d, want 3", q.SpacesAvailable())
	}
}

func TestQueueTimeoutReceive(t *testing.T) {
	q := freertos.NewQueue(10)
	defer q.Delete()

	// Receive from empty queue with no timeout
	item, ok := q.Receive(0)
	if ok {
		t.Errorf("Receive(0) from empty queue = %v, true; want nil, false", item)
	}

	// Receive with short timeout
	start := time.Now()
	item, ok = q.Receive(50)
	elapsed := time.Since(start)

	if ok {
		t.Errorf("Receive(50) from empty queue succeeded unexpectedly: %v", item)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("Receive returned too quickly: %v", elapsed)
	}
}

func TestQueueTimeoutSend(t *testing.T) {
	q := freertos.NewQueue(2)
	defer q.Delete()

	// Fill the queue
	q.Send("a", 0)
	q.Send("b", 0)

	// Send to full queue with no timeout
	if q.Send("c", 0) {
		t.Error("Send to full queue succeeded with timeout=0")
	}

	// Send with short timeout
	start := time.Now()
	ok := q.Send("c", 50)
	elapsed := time.Since(start)

	if ok {
		t.Error("Send to full queue succeeded")
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("Send returned too quickly: %v", elapsed)
	}
}

func TestQueueConcurrent(t *testing.T) {
	q := freertos.NewQueue(100)
	defer q.Delete()

	const numItems = 100
	var wg sync.WaitGroup

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			q.Send(i, freertos.MaxTimeout)
		}
	}()

	// Consumer
	received := make([]int, 0, numItems)
	var mu sync.Mutex
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			item, ok := q.Receive(freertos.MaxTimeout)
			if ok {
				mu.Lock()
				received = append(received, item.(int))
				mu.Unlock()
			}
		}
	}()

	wg.Wait()

	if len(received) != numItems {
		t.Errorf("received %d items, want %d", len(received), numItems)
	}
}

func TestQueueDelete(t *testing.T) {
	q := freertos.NewQueue(10)
	q.Send("item", 0)

	q.Delete()

	// Operations on deleted queue should fail
	if q.Send("new", 0) {
		t.Error("Send on deleted queue succeeded")
	}
	_, ok := q.Receive(0)
	if ok {
		t.Error("Receive on deleted queue succeeded")
	}
}

func TestQueueTypes(t *testing.T) {
	q := freertos.NewQueue(10)
	defer q.Delete()

	// Test different types
	q.Send(42, 0)
	q.Send("string", 0)
	q.Send(3.14, 0)
	q.Send([]int{1, 2, 3}, 0)

	item, _ := q.Receive(0)
	if item.(int) != 42 {
		t.Errorf("int: got %v", item)
	}

	item, _ = q.Receive(0)
	if item.(string) != "string" {
		t.Errorf("string: got %v", item)
	}

	item, _ = q.Receive(0)
	if item.(float64) != 3.14 {
		t.Errorf("float64: got %v", item)
	}

	item, _ = q.Receive(0)
	slice := item.([]int)
	if len(slice) != 3 {
		t.Errorf("slice: got %v", item)
	}
}
