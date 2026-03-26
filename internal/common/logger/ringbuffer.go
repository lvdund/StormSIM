package logger

import "sync"

// RingBuffer is a generic thread-safe circular buffer with update capability
type RingBuffer[T any] struct {
	entries     []T
	generations []uint64 // generation tracking for safe updates
	head        int      // next write position
	count       int      // current number of entries
	size        int      // max capacity
	nextGen     uint64   // monotonic generation counter
	mu          sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified size
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		entries:     make([]T, size),
		generations: make([]uint64, size),
		size:        size,
		nextGen:     1, // start at 1 so 0 means "never written"
	}
}

// Push adds an entry to the buffer, overwriting the oldest entry if full.
// Returns the index and generation of the pushed entry for later update.
func (rb *RingBuffer[T]) Push(entry T) (index int, generation uint64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	index = rb.head
	generation = rb.nextGen
	rb.nextGen++

	rb.entries[index] = entry
	rb.generations[index] = generation
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
	return index, generation
}

// Update updates an entry at the specified index if the generation matches.
// Returns true if update succeeded, false if the slot was overwritten (generation mismatch).
func (rb *RingBuffer[T]) Update(index int, generation uint64, entry T) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Bounds check
	if index < 0 || index >= rb.size {
		return false
	}

	// Generation check - only update if slot hasn't been overwritten
	if rb.generations[index] != generation {
		return false
	}

	rb.entries[index] = entry
	return true
}

// GetAll returns all entries in chronological order (oldest first)
func (rb *RingBuffer[T]) GetAll() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]T, rb.count)
	start := (rb.head - rb.count + rb.size) % rb.size
	for i := 0; i < rb.count; i++ {
		result[i] = rb.entries[(start+i)%rb.size]
	}
	return result
}

// GetLast returns the last n entries (most recent first)
func (rb *RingBuffer[T]) GetLast(n int) []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n > rb.count {
		n = rb.count
	}
	result := make([]T, n)
	for i := 0; i < n; i++ {
		idx := (rb.head - 1 - i + rb.size) % rb.size
		result[i] = rb.entries[idx]
	}
	return result
}

// Count returns the current number of entries in the buffer
func (rb *RingBuffer[T]) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}
