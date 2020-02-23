package prioritybuffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The buffer constructor is only needed for tests since the buffer is inlined
// into the parent struct otherwise.
func newBuffer(capacity int) *Buffer {
	b := new(Buffer)
	b.Init(capacity)
	return b
}

func TestBufferDegenerate(t *testing.T) {
	b := newBuffer(0)
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.Put(0, 0))
}

func TestBufferTrivial(t *testing.T) {
	b := newBuffer(1)
	assert.Equal(t, -1, b.Pop())

	assert.Equal(t, 0, b.Put(0, 0))
	assert.Equal(t, 1, b.length)

	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 0, b.Pop())
}

func TestBufferFavorHigherPriorityLeft(t *testing.T) {
	b := newBuffer(2)
	assert.Equal(t, -1, b.Pop())

	assert.Equal(t, 0, b.Put(0, 0))
	assert.Equal(t, 1, b.Put(0, 1))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 0, b.Pop())
	assert.Equal(t, 1, b.Pop())
	assert.Equal(t, -1, b.Pop())
}

func TestBufferFavorHigherPriorityRight(t *testing.T) {
	b := newBuffer(2)
	assert.Equal(t, -1, b.Pop())

	assert.Equal(t, 0, b.Put(0, 1))
	assert.Equal(t, 1, b.Put(0, 0))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 1, b.Pop())
	assert.Equal(t, 0, b.Pop())
	assert.Equal(t, -1, b.Pop())
}

func TestBufferDeadlineEviction(t *testing.T) {
	b := newBuffer(3)
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.EvictExpired(0))

	assert.Equal(t, 0, b.Put(0, 0))
	assert.Equal(t, 1, b.Put(1, 0))
	assert.Equal(t, 2, b.Put(2, 0))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 0, b.EvictExpired(1))
	assert.Equal(t, 1, b.EvictExpired(1))

	assert.Equal(t, -1, b.EvictExpired(1))

	assert.Equal(t, 2, b.EvictExpired(2))
	assert.Equal(t, -1, b.EvictExpired(2))
}

func TestBufferPriorityEviction(t *testing.T) {
	b := newBuffer(3)
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.EvictExpired(0))

	assert.Equal(t, 0, b.Put(0, 2))
	assert.Equal(t, 1, b.Put(0, 0))
	assert.Equal(t, 2, b.Put(0, 1))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, -1, b.EvictLowerPriority(3))
	assert.Equal(t, -1, b.EvictLowerPriority(2))
	assert.Equal(t, b.minPriorities, b.coMinPriorities)
	assert.Equal(t, 0, b.EvictLowerPriority(1))

	assert.Equal(t, 2, b.EvictLowerPriority(0))
	assert.Equal(t, -1, b.EvictLowerPriority(0))
	assert.Equal(t, 1, b.Pop())
	assert.Equal(t, -1, b.EvictLowerPriority(0))
	assert.Equal(t, -1, b.Pop())
}
