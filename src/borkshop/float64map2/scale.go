package float64map2

// NewScale scales a map horizontally and vertically, stretching or shrinking
// along each axis.
func NewScale(source Map, scale float64) Scale {
	return Scale{source: source, scale: scale}
}

// Scale is a map that has been scaled.
type Scale struct {
	source Map
	scale  float64
}

// Eval2 gives the value at a coordinate.
func (m Scale) Eval2(x, y float64) float64 {
	return m.source.Eval2(x*m.scale, y*m.scale)
}
