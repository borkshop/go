package modspace

import "image"

// Space is a space with the given height and width
type Space image.Point

// Add adds two points, wrapping around the space at the edges.
func (s Space) Add(a, b image.Point) image.Point {
	return image.Pt((a.X+b.X)&(s.X-1), (a.Y+b.Y)&(s.Y-1))
}
