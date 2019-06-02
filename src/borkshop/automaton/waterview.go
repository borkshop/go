package main

import (
	"image"
)

type WaterView struct {
	automaton *Automaton
	image     *image.RGBA
}

func NewWaterView(automaton *Automaton) *WaterView {
	image := image.NewRGBA(automaton.rect)
	return &WaterView{
		automaton: automaton,
		image:     image,
	}
}

func (v *WaterView) Draw(screen *image.RGBA, rect image.Rectangle) {
	// TODO offset point
	drawScale(v.image, v.automaton.water, v.automaton.waterStats, 0, 255, v.automaton.points)
	drawScreen(screen, rect, v.image)
}
