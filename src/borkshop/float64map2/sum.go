package float64map2

// Sum is a map where each value is the sum of the values of other maps.
type Sum []Map

// Eval2 gives the value at a coordinate.
func (s Sum) Eval2(x, y float64) float64 {
	var z float64
	for i := 0; i < len(s); i++ {
		z += s[i].Eval2(x, y)
	}
	return z
}
