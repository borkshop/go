// Package prioritybuffer provides a buffer that indexes entities by priority
// and deadline for fast eviction of expired or low priority entities and
// consumpution of high priority entities.
package prioritybuffer

// This is a generalization of the buffer I wrote for Uber's YARPC under the
// MIT license.
// https://github.com/yarpc/yarpc-go/pull/1881
// - kriskowal

import (
	"borkshop/heapuint64int"
	"borkshop/jotint"
	"borkshop/repeatuint64"
	"borkshop/swapintint"
	"borkshop/typeuint64"
)

// Buffer is a priority buffer indexed by priority and deadline.
type Buffer struct {
	capacity int
	length   int

	// columns
	deadlines  []uint64
	priorities []uint64

	// indexes
	entities        []int
	coEntities      []int
	minPriorities   []int
	coMinPriorities []int
	maxPriorities   []int
	coMaxPriorities []int
	minDeadlines    []int
	coMinDeadlines  []int
}

// Init initializes a buffer for a given capacity.
//
// Buffers are intended to be inlined in another type that tracks other columns
// of data for each entity.
func (b *Buffer) Init(capacity int) {
	b.capacity = capacity
	b.deadlines = repeatuint64.Repeat(capacity, capacity, typeuint64.Max)
	b.priorities = repeatuint64.Repeat(capacity, capacity, typeuint64.Max)
	b.entities = jotint.Jot(capacity)
	b.coEntities = jotint.Jot(capacity)
	b.minPriorities = jotint.Jot(capacity)
	b.coMinPriorities = jotint.Jot(capacity)
	b.maxPriorities = jotint.Jot(capacity)
	b.coMaxPriorities = jotint.Jot(capacity)
	b.minDeadlines = jotint.Jot(capacity)
	b.coMinDeadlines = jotint.Jot(capacity)
}

// IsEmpty indicates that no entities are allocated.
func (b *Buffer) IsEmpty() bool {
	return b.length == 0
}

// IsFull indicates that no further entities can be added to the buffer until one
// is removed.
func (b *Buffer) IsFull() bool {
	return b.length >= b.capacity
}

// Put adds an entity with the given deadline and priority (lower priorities
// have precedence), returns the allocated entity index, or -1 if the buffer is
// full.
func (b *Buffer) Put(deadline uint64, priority uint64) int {
	if b.length >= b.capacity {
		return -1
	}

	// Choose an entity index directly on the partition of the free entity
	// index.
	i := b.entities[b.length]
	// Move the entity to the partition on all corresponding indexes.
	swapintint.Swap(b.maxPriorities, b.coMaxPriorities, b.coMaxPriorities[i], b.length)
	swapintint.Swap(b.minPriorities, b.coMinPriorities, b.coMinPriorities[i], b.length)
	swapintint.Swap(b.minDeadlines, b.coMinDeadlines, b.coMinDeadlines[i], b.length)
	// Shift the partition, introducing the entity to every index.
	b.length++

	// Adjust indexes:

	// Assuming all deadlines are positive in the Unix epoch.
	// Time travellers, please do not use this algorithm before 1970.
	b.deadlines[i] = deadline
	heapuint64int.FixUp(heapuint64int.Min, b.deadlines, b.minDeadlines, b.coMinDeadlines, i)

	b.priorities[i] = priority
	heapuint64int.FixUp(heapuint64int.Min, b.priorities, b.minPriorities, b.coMinPriorities, i)
	heapuint64int.FixUp(heapuint64int.Max, b.priorities, b.maxPriorities, b.coMaxPriorities, i)

	return i
}

// Pop removes and returns the index for the highest priority entity in the
// buffer.
func (b *Buffer) Pop() int {
	if b.length == 0 {
		return -1
	}

	// Index of highest priority () entity.
	i := b.minPriorities[0]

	b.evict(i)

	return i
}

// Top returns the index of the highest priority entity in the buffer.
func (b *Buffer) Top() int {
	if b.length == 0 {
		return -1
	}

	// Index of highest priority () entity.
	return b.minPriorities[0]
}

// Priority returns the priority of an entity.
func (b *Buffer) Priority(index int) uint64 {
	return b.priorities[index]
}

// SetPriority adjusts the priority of an entity.
func (b *Buffer) SetPriority(index int, priority uint64) {
	b.priorities[index] = priority

	heapuint64int.Fix(heapuint64int.Min, b.length, b.priorities, b.minPriorities, b.coMinPriorities, index)
	heapuint64int.Fix(heapuint64int.Max, b.length, b.priorities, b.maxPriorities, b.coMaxPriorities, index)
}

// Evict removes and returns the index of the entity with the earliest expired
// deadline, or -1 if no entity has expired.
func (b *Buffer) EvictExpired(now uint64) int {
	if b.length == 0 {
		return -1
	}

	// Index of next entity to expire.
	i := b.minDeadlines[0]

	if b.deadlines[i] > now {
		return -1
	}

	b.evict(i)

	return i
}

// EvictLowerPriority removes and returns the index of the entity in the buffer
// that has a lower priority (lower priorities are numerically higher).
func (b *Buffer) EvictLowerPriority(priority uint64) int {
	if b.length == 0 {
		return -1
	}

	// Index of of lowest priority entity.
	i := b.maxPriorities[0]

	// Favor keeping an entity over replacing with an equivalent, for the sake
	// of churn.
	if b.priorities[i] <= priority {
		return -1
	}

	b.evict(i)

	return i
}

// evict removes an entity from the buffer, frees its index for a future
// entity, and adjusts the internal heaps.
func (b *Buffer) evict(i int) {
	// Reset values
	b.deadlines[i] = typeuint64.Max
	b.priorities[i] = typeuint64.Max

	// One less entity.
	// Move partition first, so we can use the new length as the destination
	// index for swaps.
	b.length--

	// Move the selected entity out beyond the horizon.
	swapintint.Swap(b.entities, b.coEntities, b.coEntities[i], b.length)

	// Similarly, swap entities in each index, then fix the heaps.

	swapintint.Swap(b.minDeadlines, b.coMinDeadlines, b.coMinDeadlines[i], b.length)
	heapuint64int.Fix(heapuint64int.Min, b.length, b.deadlines, b.minDeadlines, b.coMinDeadlines, b.length)

	swapintint.Swap(b.minPriorities, b.coMinPriorities, b.coMinPriorities[i], b.length)
	heapuint64int.Fix(heapuint64int.Min, b.length, b.priorities, b.minPriorities, b.coMinPriorities, b.length)

	swapintint.Swap(b.maxPriorities, b.coMaxPriorities, b.coMaxPriorities[i], b.length)
	heapuint64int.Fix(heapuint64int.Max, b.length, b.priorities, b.maxPriorities, b.coMaxPriorities, b.length)
}
