package main

import (
	"image"
	"image/color"
	"image/draw"
)

var (
	black = color.Black
	brown = color.RGBA{205, 178, 117, 0}
	blue  = color.RGBA{20, 20, 64, 0}
)

type MapView struct {
	automaton *Automaton
	img       *image.RGBA
	earth     *image.RGBA
	water     *image.RGBA
}

func NewMapView(automaton *Automaton) *MapView {
	img := image.NewRGBA(automaton.rect)
	earth := image.NewRGBA(automaton.rect)
	water := image.NewRGBA(automaton.rect)
	return &MapView{
		automaton: automaton,
		img:       img,
		earth:     earth,
		water:     water,
	}
}

func (v *MapView) Name() string {
	return "map"
}

func (v *MapView) Draw(screen *image.RGBA, rect image.Rectangle) {
	draw.Draw(v.img, v.img.Rect, &image.Uniform{black}, image.ZP, draw.Over)

	drawAlpha(v.earth, v.automaton.earth, v.automaton.earthStats, 0, 255, v.automaton.points)
	draw.DrawMask(v.img, v.img.Rect, &image.Uniform{brown}, image.ZP, v.earth, image.ZP, draw.Over)

	drawWater(v.water, v.automaton.water, v.automaton.waterStats, 128, 255, v.automaton.significantWater, v.automaton.points)
	draw.DrawMask(v.img, v.img.Rect, &image.Uniform{blue}, image.ZP, v.water, image.ZP, draw.Over)

	drawScreen(screen, rect, v.img)
}

func drawWater(dst *image.RGBA, values []int64, stats Stats64, min, max, sig int64, points []image.Point) {
	for i := 0; i < len(values); i++ {
		value := values[i]
		var alpha uint8
		if value > sig {
			alpha = uint8(min + stats.Project(value, max-min))
		}
		pt := points[i]
		dst.SetRGBA(pt.X, pt.Y, color.RGBA{0, 0, 0, alpha})
	}
}
