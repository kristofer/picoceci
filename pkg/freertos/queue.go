package freertos

// Queue provides thread-safe inter-task communication.
// On FreeRTOS, this wraps xQueueCreate/xQueueSend/xQueueReceive.
// On desktop, this uses a Go channel.
type Queue interface {
	// Send adds an item to the back of the queue.
	// Returns true if sent successfully, false on timeout.
	// timeoutMs: max wait time in milliseconds (0 = no wait, MaxTimeout = forever)
	Send(item interface{}, timeoutMs uint32) bool

	// SendToFront adds an item to the front of the queue.
	SendToFront(item interface{}, timeoutMs uint32) bool

	// Receive removes and returns an item from the front of the queue.
	// Returns (item, true) if received, (nil, false) on timeout.
	Receive(timeoutMs uint32) (interface{}, bool)

	// Peek returns the front item without removing it.
	Peek(timeoutMs uint32) (interface{}, bool)

	// Count returns the number of items currently in the queue.
	Count() int

	// SpacesAvailable returns the number of free slots.
	SpacesAvailable() int

	// Capacity returns the maximum queue size.
	Capacity() int

	// Delete releases queue resources.
	Delete()
}

// MaxTimeout represents an infinite wait.
const MaxTimeout uint32 = 0xFFFFFFFF

// NewQueue creates a queue with the specified capacity.
func NewQueue(capacity int) Queue {
	return newQueue(capacity)
}
