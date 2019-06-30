package main

import (
	"image"
)

type EarthView struct {
	automaton *Automaton
}

func NewEarthView(automaton *Automaton) *EarthView {
	return &EarthView{automaton: automaton}
}

func (v *EarthView) Name() string {
	return "earth"
}

func (v *EarthView) Draw(screen *image.RGBA, rect image.Rectangle) {
	// TODO offset point
	drawScale(screen, v.automaton.earth, v.automaton.earthStats, 0, 255, v.automaton.points)
}
