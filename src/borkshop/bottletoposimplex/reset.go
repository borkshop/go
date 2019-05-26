package bottletoposimplex

import (
	"borkshop/bottle"
	"borkshop/bottlefloat64map2"
	"borkshop/float64map2"
	"borkshop/hilbert"

	"github.com/ojrac/opensimplex-go"
)

// New creates a map resetter that uses a sum series of opensimplex noise to
// set the elevation of the earth columns.
func New(scale int) bottle.Resetter {
	width := float64(scale)
	height := width
	scales := []struct {
		seed                       int64
		terrainScale, simplexScale float64
	}{
		{1, 256, 1.0 / 256},
		{2, 64, 1.0 / 64},
		{3, 16, 1.0 / 32},
		{4, 32, 1.0 / 16},
		{5, 16, 1.0 / 8},
		{6, 4, 1.0 / 4},
		{7, 2, 1.0 / 2},
	}
	noises := make([]float64map2.Map, 0, len(scales))
	for _, s := range scales {
		os := opensimplex.New(s.seed)
		ss := float64map2.NewScale(os, s.simplexScale)
		ts := float64map2.NewTesselation(ss, width, height)
		as := float64map2.NewAmplify(ts, s.terrainScale)
		noises = append(noises, as)
	}

	return bottlefloat64map2.NewResetter(
		hilbert.Scale(scale),
		float64map2.Sum(noises),
	)
}
