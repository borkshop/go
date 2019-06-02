package main

import (
	"image"
)

func drawScreen(screen *image.RGBA, rect image.Rectangle, bg *image.RGBA) {
	var pt image.Point
	for pt.Y = rect.Min.Y; pt.Y < rect.Max.Y; pt.Y++ {
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X++ {
			if !pt.In(screen.Rect) {
				continue
			}
			if !pt.In(rect) {
				continue
			}
			screen.SetRGBA(pt.X, pt.Y, bg.RGBAAt(pt.X, pt.Y))
		}
	}
}
