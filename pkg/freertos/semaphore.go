package freertos

// SemaphoreKind specifies the type of semaphore.
type SemaphoreKind uint8

const (
	// SemaphoreBinary is a binary semaphore (0 or 1).
	SemaphoreBinary SemaphoreKind = iota

	// SemaphoreCounting is a counting semaphore (0 to maxCount).
	SemaphoreCounting

	// SemaphoreMutex is a mutex with priority inheritance.
	SemaphoreMutex
)

// Semaphore provides synchronization between tasks.
// On FreeRTOS, this wraps xSemaphoreCreate*/xSemaphoreTake/xSemaphoreGive.
// On desktop, this uses channels or sync.Mutex.
type Semaphore interface {
	// Take acquires the semaphore.
	// Returns true if acquired, false on timeout.
	// timeoutMs: max wait time in milliseconds (0 = no wait, MaxTimeout = forever)
	Take(timeoutMs uint32) bool

	// Give releases the semaphore.
	// Returns true if released successfully.
	Give() bool

	// GetCount returns the current count (for counting semaphores).
	GetCount() int

	// Delete releases semaphore resources.
	Delete()
}

// NewBinarySemaphore creates a binary semaphore.
// Starts in the "taken" state - must Give before Take will succeed.
func NewBinarySemaphore() Semaphore {
	return newSemaphore(SemaphoreBinary, 1)
}

// NewCountingSemaphore creates a counting semaphore.
// maxCount: maximum count value
// initialCount: starting count (usually 0 or maxCount)
func NewCountingSemaphore(maxCount, initialCount int) Semaphore {
	s := newSemaphore(SemaphoreCounting, maxCount)
	// Set initial count by giving
	for i := 0; i < initialCount; i++ {
		s.Give()
	}
	return s
}

// NewMutex creates a mutex semaphore.
// Mutexes support priority inheritance to prevent priority inversion.
func NewMutex() Semaphore {
	s := newSemaphore(SemaphoreMutex, 1)
	// Mutex starts in "given" state
	s.Give()
	return s
}
