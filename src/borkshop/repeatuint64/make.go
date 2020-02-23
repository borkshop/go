// Package repeatuint64 provides a function for creating slices of repeated
// uint64 values.
package repeatuint64

// Repeat creates a slice of the repeated value.
func Repeat(length, capacity int, val uint64) []uint64 {
	slice := make([]uint64, length, capacity)
	for i := 0; i < length; i++ {
		slice[i] = val
	}
	return slice
}
