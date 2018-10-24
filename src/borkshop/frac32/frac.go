// Package frac32 provides fixed-width representations of fractions in the range
// of [0, 1] (mantissas).
package frac32

import "math/rand"

// Frac is a 32 bit mantissa, suitable for representing fixed point
// "percentages" or "fractions" in the range from [0, 1].
//
// The interval of represented values is inclusive of both the lower and upper
// bound.
// Methods of the fraction ensure that the entire range is expressible and
// preserves the identity that 1*1=1 and 0*n=0.
type Frac uint32

const (
	// Whole is a fraction representing 1.0.
	Whole Frac = 0xffffffff >> iota
	// Half is a fraction representing approximately 0.5.
	Half
	// Quarter is a fraction representing approximately 0.25.
	Quarter
	// Epsilon is the smallest discrete fraction of a fraction.
	Epsilon = 1
	// Zero is a zero as a fraction.
	Zero Frac = 0
)

// Mul multiplies a pair of fractions.
func (frac Frac) Mul(other Frac) Frac {
	var acc uint64
	acc = uint64(frac) + 1
	acc *= uint64(other) + 1
	return Frac((acc - 1) >> 32)
}

// Project projects a fraction into a range between two other fractions.
func (frac Frac) Project(min, max Frac) Frac {
	if min > max {
		max, min = min, max
	}
	return (max - min).Mul(frac) + min
}

// Float32 converts a fraction to a 32 bit floating point number.
func (frac Frac) Float32() float32 {
	return float32(frac) / float32(Whole)
}

// Float64 converts a fraction to a 64 bit floating point number.
func (frac Frac) Float64() float64 {
	return float64(frac) / float64(Whole)
}

// Random returns a random fraction between [0, 1].
func Random() Frac {
	return Frac(rand.Uint32())
}

// Per returns a fractional value with the given numerator and denominator
// like 1/4 or 3/5.
func Per(num, den int) Frac {
	acc := uint64(Whole)
	acc /= uint64(den)
	acc *= uint64(num)
	return Frac(acc)
}
