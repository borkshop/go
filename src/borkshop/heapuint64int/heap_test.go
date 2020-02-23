package heapuint64int

import (
	"borkshop/jotint"
	"borkshop/jotuint64"
	"borkshop/typeuint64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeap(t *testing.T) {
	size := 10

	values := jotuint64.Jot(size)
	heap := jotint.Jot(size)
	coHeap := jotint.Jot(size)

	t.Run("down", func(t *testing.T) {
		for i := 0; i < size-1; i++ {
			values[i] = typeuint64.Max
			Fix(Min, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(i+1), values[heap[0]])
		}

		values[size-1] = typeuint64.Max
		Fix(Min, size, values, heap, coHeap, size-1)
		assert.Equal(t, typeuint64.Max, values[heap[0]])
	})

	t.Run("up", func(t *testing.T) {
		for i := size - 1; i > 0; i-- {
			values[i] = uint64(i)
			Fix(Min, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(i), values[heap[0]])
		}
	})

	t.Run("up_with_duplicates", func(t *testing.T) {
		for i := 0; i < size-1; i++ {
			values[i] = typeuint64.Max
			Fix(Min, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(i+1), values[heap[0]])
		}

		for i := size - 1; i > 0; i-- {
			// Dividing by half results in 0, 0, 1, 1, &c.
			// So, the second sift runs into its doppleganger and exercises the
			// case where the parent is not less than itself.
			values[i] = uint64(size / 2)
			Fix(Min, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(size/2), values[heap[0]])
		}
	})
}
