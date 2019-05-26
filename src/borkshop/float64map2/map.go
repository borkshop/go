// Package floatmap64 provides common transforms for two dimensional maps of 64
// bit floating point values.
package float64map2

// Map is a two dimensional field of values, using 64 bit floats.
type Map interface {
	Eval2(x, y float64) float64
}
