package stencil

import (
	"borkshop/hilbert"
	"image"
)

func WriteHilbertPoints(dst []image.Point, length int) {
	var pt image.Point
	for pt.Y = 0; pt.Y < length; pt.Y++ {
		for pt.X = 0; pt.X < length; pt.X++ {
			i := hilbert.Encode(pt, length)
			dst[i] = pt
		}
	}
}

func RasterHilbertInt64Vector(dst []int64, src []int64, length int) {
	var pt image.Point
	var i int
	for pt.Y = 0; pt.Y < length; pt.Y++ {
		for pt.X = 0; pt.X < length; pt.X++ {
			h := hilbert.Encode(pt, length)
			dst[i] = src[h]
			i++
		}
	}
}

func WriteHilbertStencil3Table(dst [][2]int, length int) {
	area := len(dst)
	for i := 0; i < area; i++ {
		pt := hilbert.Decode(i, length)
		dst[i][0] = hilbert.Encode(image.Point{X: (pt.X + 1), Y: pt.Y}, length)
		dst[i][1] = hilbert.Encode(image.Point{X: pt.X, Y: (pt.Y + 1)}, length)
	}
}

func WriteHilbertStencil5Table(dst [][4]int, length int) {
	area := len(dst)
	for i := 0; i < area; i++ {
		pt := hilbert.Decode(i, length)
		dst[i][0] = hilbert.Encode(image.Point{X: pt.X + 1, Y: pt.Y}, length)
		dst[i][1] = hilbert.Encode(image.Point{X: pt.X, Y: pt.Y + 1}, length)
		dst[i][2] = hilbert.Encode(image.Point{X: pt.X - 1, Y: pt.Y}, length)
		dst[i][3] = hilbert.Encode(image.Point{X: pt.X, Y: pt.Y - 1}, length)
	}
}

func WriteHilbertStencil9Table(dst [][8]int, length int) {
	area := len(dst)
	for i := 0; i < area; i++ {
		pt := hilbert.Decode(i, length)
		dst[i][0] = hilbert.Encode(image.Point{X: pt.X + 1, Y: pt.Y}, length)
		dst[i][1] = hilbert.Encode(image.Point{X: pt.X, Y: pt.Y + 1}, length)
		dst[i][2] = hilbert.Encode(image.Point{X: pt.X - 1, Y: pt.Y}, length)
		dst[i][3] = hilbert.Encode(image.Point{X: pt.X, Y: pt.Y - 1}, length)
		dst[i][4] = hilbert.Encode(image.Point{X: pt.X + 1, Y: pt.Y + 1}, length)
		dst[i][5] = hilbert.Encode(image.Point{X: pt.X - 1, Y: pt.Y + 1}, length)
		dst[i][6] = hilbert.Encode(image.Point{X: pt.X - 1, Y: pt.Y - 1}, length)
		dst[i][7] = hilbert.Encode(image.Point{X: pt.X + 1, Y: pt.Y - 1}, length)
	}
}
