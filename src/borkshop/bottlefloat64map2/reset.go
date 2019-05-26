package bottlefloat64map2

import (
	"borkshop/bottle"
	"borkshop/float64map2"
	"borkshop/hilbert"
	"image"
)

// NewResetter creates a generation setter that initializes the levels of the
// terrain to the values from the given floating point map.
func NewResetter(scale hilbert.Scale, source float64map2.Map) Resetter {
	return Resetter{scale: scale, source: source}
}

// Resetter sets the levels of terrain to the values from a floating point map.
type Resetter struct {
	scale  hilbert.Scale
	source float64map2.Map
}

func (r Resetter) Reset(gen *bottle.Generation) {
	width := int(r.scale)
	height := int(r.scale)

	var pt image.Point
	for pt.Y = 0; pt.Y < height; pt.Y++ {
		for pt.X = 0; pt.X < width; pt.X++ {
			cel := &gen.Grid[r.scale.Encode(pt)]
			cel.Earth = int(r.source.Eval2(float64(pt.X), float64(pt.Y)))
		}
	}
}
