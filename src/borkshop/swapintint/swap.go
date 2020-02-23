// Package swapintint provides a Swap function for arrays of int values and a
// reverse look-up table of ints.
package swapintint

// Swap swaps the values at a pair of indexes and updates the reverse-lookup
// table.
func Swap(values, coValues []int, i, j int) {
	a, b := values[i], values[j]
	values[i], values[j] = values[j], values[i]
	coValues[a], coValues[b] = j, i
}
