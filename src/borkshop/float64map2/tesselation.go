package float64map2

// NewTesselation takes a map and wraps it horizontally and vertically so it
// appears seamless accross all edges.
func NewTesselation(source Map, width, height float64) Tesselation {
	return Tesselation{
		source: source,
		width:  width,
		height: height,
	}
}

// Tesselation is a map that's wrapped horizontally and vertically.
type Tesselation struct {
	source        Map
	width, height float64
}

// Eval2 gives the value at a coordinate.
func (t Tesselation) Eval2(x, y float64) float64 {
	a := t.source.Eval2(float64(x), float64(y))
	b := t.source.Eval2(float64(x-t.width), float64(y))
	ab := a*(1.0-float64(x)/t.width) + b*(float64(x)/t.width)
	c := t.source.Eval2(float64(x), float64(y-t.height))
	d := t.source.Eval2(float64(x-t.width), float64(y-t.height))
	cd := c*(1.0-float64(x)/t.width) + d*(float64(x)/t.width)
	return ab*(1.0-float64(y)/t.height) + cd*float64(y)/t.height
}
