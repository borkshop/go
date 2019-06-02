package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

type EarthView struct {
	automaton *Automaton
	image     *image.RGBA
}

func NewAnansiEarthView(automaton *Automaton) *EarthView {
	image := image.NewRGBA(automaton.rect)
	return &EarthView{
		automaton: automaton,
		image:     image,
	}
}

func (v *EarthView) Draw(screen *anansi.Screen, rect ansi.Rectangle) {
	// TODO offset point
	drawScale(v.image, v.automaton.earth, v.automaton.earthStats, 0, 255, v.automaton.points)
	drawAnansi(screen, rect, v.image)
}
