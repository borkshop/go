package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

type WaterView struct {
	automaton *Automaton
	image     *image.RGBA
}

func NewAnansiWaterView(automaton *Automaton) *WaterView {
	image := image.NewRGBA(automaton.rect)
	return &WaterView{
		automaton: automaton,
		image:     image,
	}
}

func (v *WaterView) Draw(screen *anansi.Screen, rect ansi.Rectangle) {
	// TODO offset point
	drawScale(v.image, v.automaton.water, v.automaton.waterStats, 0, 255, v.automaton.points)
	drawAnansi(screen, rect, v.image)
}
