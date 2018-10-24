// Package xorshiftstar implements the xorshift* pseudorandom number generator.
//
// https://en.wikipedia.org/wiki/Xorshift
package xorshiftstar

import "math/rand"

// Source is a xorshiftstar random number generator.
type Source struct {
	state uint64
}

var (
	_ rand.Source = (*Source)(nil)
)

// New returns a new random number generator for the given seed.
func New(seed int) *Source {
	return &Source{
		state: uint64(seed + 1),
	}
}

// Seed seeds the random number generator.
func (r *Source) Seed(seed int64) {
	r.state = uint64(seed)
}

// Uint64 returns a random number.
func (r *Source) Uint64() uint64 {
	state := r.state + 1442695040888963407
	state ^= state >> 12
	state ^= state << 25
	state ^= state >> 27
	r.state = state
	return state * 6364136223846793005
}

// Int63 returns a random number.
func (r *Source) Int63() int64 {
	state := r.state
	state ^= state >> 12
	state ^= state << 25
	state ^= state >> 27
	r.state = state
	return int64((state * 6364136223846793005) >> 1)
}
