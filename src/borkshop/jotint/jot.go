// Package jotint generates sequential integer slices.
package jotint

// Jot produces a slice of sequential integers starting with 0.
func Jot(num int) []int {
	slice := make([]int, num)
	for i := 0; i < num; i++ {
		slice[i] = i
	}
	return slice
}
