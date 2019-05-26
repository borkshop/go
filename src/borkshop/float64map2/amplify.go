package float64map2

// NewAmplify returns a map where every value is multiplied by a value.
func NewAmplify(source Map, value float64) Amplify {
	return Amplify{source: source, value: value}
}

// Amplify multiplies the magnitude of the value at every coordinate.
type Amplify struct {
	source Map
	value  float64
}

// Eval2 gives the value at a coordinate.
func (m Amplify) Eval2(x, y float64) float64 {
	return m.value * m.source.Eval2(x, y)
}
