package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

type AnansiEarthView struct {
	automaton *Automaton
	image     *image.RGBA
}

func NewAnansiEarthView(automaton *Automaton) *AnansiEarthView {
	image := image.NewRGBA(automaton.rect)
	return &AnansiEarthView{
		automaton: automaton,
		image:     image,
	}
}

func (v *AnansiEarthView) Draw(screen *anansi.Screen, rect ansi.Rectangle) {
	// TODO offset point
	drawScale(v.image, v.automaton.earth, v.automaton.earthStats, 0, 255, v.automaton.points)
	drawAnansi(screen, rect, v.image)
}
