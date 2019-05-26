package bottleflood

import (
	"borkshop/bottle"
	"borkshop/hilbert"
	"image"
)

func New(scale hilbert.Scale, level int) Resetter {
	return Resetter{scale: scale, level: level}
}

// Resetter sets the levels of terrain to the values from a floating point map.
type Resetter struct {
	scale hilbert.Scale
	level int
}

func (r Resetter) Reset(gen *bottle.Generation) {
	width := int(r.scale)
	height := int(r.scale)

	var pt image.Point
	for pt.Y = 0; pt.Y < height; pt.Y++ {
		for pt.X = 0; pt.X < width; pt.X++ {
			cel := &gen.Grid[r.scale.Encode(pt)]
			cel.Water = r.level
		}
	}
}
