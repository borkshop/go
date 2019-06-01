package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

func drawAnansi(screen *anansi.Screen, rect ansi.Rectangle, bg *image.RGBA) {
	grid := screen.Grid
	var pt ansi.Point
	for pt.Y = rect.Min.Y; pt.Y < rect.Max.Y; pt.Y++ {
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X++ {
			o, ok := grid.CellOffset(pt)
			if !ok {
				continue
			}
			if !pt.In(rect) {
				continue
			}
			ipt := pt.ToImage()
			ipt.Y *= 2
			r, g, b, a := bg.At(ipt.X, ipt.Y).RGBA()
			bg := ansi.RGBA(r, g, b, a)
			// grid.Rune[o] = '·'
			grid.Attr[o] = bg.BG()
		}
	}
}
