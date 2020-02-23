// Package jotint generates sequential uint64 slices.
package jotuint64

// Jot produces a slice of sequential integers starting with 0.
func Jot(num int) []uint64 {
	slice := make([]uint64, num)
	for i := 0; i < num; i++ {
		slice[i] = uint64(i)
	}
	return slice
}
