// Package heapuint64int provides a heap fixer with uint64 values and integer
// indexes.
//
// This heap implementation relies on parallel slices representing columns of
// data to simplify managing multiple indexes on tables.
package heapuint64int

import "borkshop/swapintint"

// Top indicates whether the Min or Max values is on top of a heap.
type Top bool

const (
	Min Top = true
	Max     = false
)

// Fix sifts a min or max heap after the value at an index changes.
//
// Updates both the heap and coHeap such that the heap contains the indexes of
// values and the coHeap contains the indexes of the heap for the values at the
// same positions.
// Maintaining heap and coHeap as parallel arrays simplifies the use of
// multiple indexes on the same values, or values from entities.
//
// The length refers to the working portion of the values, heap, and coHeap.
func Fix(top Top, length int, values []uint64, heap, coHeap []int, index int) {
	FixUp(top, values, heap, coHeap, index)
	FixDown(top, length, values, heap, coHeap, index)
}

// FixUp sifts a min or max heap after the value at an index changes, that is
// decreasing for min heaps and increasing for max heaps.
//
// Updates both the heap and coHeap such that the heap contains the indexes of
// values and the coHeap contains the indexes of the heap for the values at the
// same positions.
// Maintaining heap and coHeap as parallel arrays simplifies the use of
// multiple indexes on the same values, or values from entities.
//
// Fixing toward the top of the heap does not require the length of the heap
// since it will only sift values that are between the top and the indexed
// value.
func FixUp(top Top, values []uint64, heap, coHeap []int, index int) {
	value := values[index]
	i := coHeap[index]
	for i > 0 {
		pi := (i+1)/2 - 1
		parent := values[heap[pi]]
		if parent > value == top {
			swapintint.Swap(heap, coHeap, i, pi)
			i = pi
		} else {
			break
		}
	}
}

// FixDown sifts a min or max heap after the value at an index changes, that is
// increasing for min heaps and decreasing for max heaps.
//
// Updates both the heap and coHeap such that the heap contains the indexes of
// values and the coHeap contains the indexes of the heap for the values at the
// same positions.
// Maintaining heap and coHeap as parallel arrays simplifies the use of
// multiple indexes on the same values, or values from entities.
//
// The length refers to the working portion of the values, heap, and coHeap.
func FixDown(top Top, length int, values []uint64, heap, coHeap []int, index int) {
	value := values[index]
	i := coHeap[index]

	for {
		ri := (i + 1) * 2
		li := ri - 1
		si := -1 // Sentinel value indicates neither child is misplaced.

		var left uint64
		// Consider left child, if one eixsts.
		if li < length {
			left = values[heap[li]]
			if left < value == top {
				si = li
			}
		}

		// Consider right child, if one exists
		if ri < length {
			right := values[heap[ri]]
			if si >= 0 {
				// Favor the right child over the left child only if it is
				// even more displaced.
				if right < left == top {
					si = ri
				}
			} else {
				if right < value == top {
					si = ri
				}
			}
		}

		// All done if both children are already where they ought to be.
		if si < 0 {
			break
		}

		// Swap either left or right, whichever was farther out of place.
		swapintint.Swap(heap, coHeap, i, si)
		// Continue floating from the swapped child.
		i = si
	}
}
