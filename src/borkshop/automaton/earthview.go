package main

import (
	"image"
)

type EarthView struct {
	automaton *Automaton
	image     *image.RGBA
}

func NewEarthView(automaton *Automaton) *EarthView {
	image := image.NewRGBA(automaton.rect)
	return &EarthView{
		automaton: automaton,
		image:     image,
	}
}

func (v *EarthView) Draw(screen *image.RGBA, rect image.Rectangle) {
	// TODO offset point
	drawScale(v.image, v.automaton.earth, v.automaton.earthStats, 0, 255, v.automaton.points)
	drawScreen(screen, rect, v.image)
}
