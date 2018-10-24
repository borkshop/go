package frac32_test

import (
	"testing"

	"borkshop/frac32"

	"github.com/stretchr/testify/assert"
)

func TestConsts(t *testing.T) {
	assert.Equal(t, frac32.Zero.Float64(), 0.0)
	assert.Equal(t, frac32.Whole.Float64(), 1.0)
	assertInDelta(t, frac32.Half, 0.5)
	assertInDelta(t, frac32.Quarter, 0.25)
}

func TestMul(t *testing.T) {
	assertInDelta(t, frac32.Whole.Mul(frac32.Half), 0.5)
	assertInDelta(t, frac32.Half.Mul(frac32.Half), 0.25)
	assertInDelta(t, frac32.Quarter.Mul(frac32.Half), 0.125)
	assertInDelta(t, frac32.Quarter.Mul(frac32.Quarter), 0.0625)
}

func TestPer(t *testing.T) {
	assertInDelta(t, frac32.Per(1, 2), 0.5)
	assertInDelta(t, frac32.Per(1, 4), 0.25)
	assertInDelta(t, frac32.Per(1, 8), 0.125)
	assertInDelta(t, frac32.Per(1, 16), 0.0625)

	assertInDelta(t, frac32.Per(1, 3), 1.0/3.0)
	assertInDelta(t, frac32.Per(2, 3), 2.0/3.0)

	assertInDelta(t, frac32.Per(1, 10000), 1.0/10000.0)
}

func TestProject(t *testing.T) {
	assertInDelta(t, frac32.Half.Project(frac32.Per(1, 3), frac32.Per(2, 3)), 0.5)
	assertInDelta(t, frac32.Half.Project(frac32.Per(2, 3), frac32.Per(1, 3)), 0.5)
}

func assertInDelta(t *testing.T, frac frac32.Frac, float float64) {
	if !assert.True(t, (frac+frac32.Epsilon).Float64() >= float) {
		t.Logf("Expected %v to be more than %v\n", (frac + frac32.Epsilon).Float64(), float)
	}
	if !assert.True(t, (frac-frac32.Epsilon).Float64() <= float) {
		t.Logf("Expected %v to be less than %v\n", frac.Float64(), float)
	}
}
