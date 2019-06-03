package main

import (
	"image"
	"image/color"
)

func drawScale(dst *image.RGBA, values []int64, stats Stats64, min, max int64, points []image.Point) {
	for i := 0; i < len(values); i++ {
		v := uint8(min + stats.Project(values[i], max-min))
		pt := points[i]
		dst.SetRGBA(pt.X, pt.Y, color.RGBA{v, v, v, v})
	}
}

func drawAlpha(dst *image.RGBA, values []int64, stats Stats64, min, max int64, points []image.Point) {
	for i := 0; i < len(values); i++ {
		v := uint8(min + stats.Project(values[i], max-min))
		pt := points[i]
		dst.SetRGBA(pt.X, pt.Y, color.RGBA{0, 0, 0, v})
	}
}
