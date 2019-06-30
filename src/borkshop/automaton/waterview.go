package main

import (
	"image"
)

type WaterView struct {
	automaton *Automaton
}

func NewWaterView(automaton *Automaton) *WaterView {
	return &WaterView{automaton: automaton}
}

func (v *WaterView) Name() string {
	return "water"
}

func (v *WaterView) Draw(screen *image.RGBA, rect image.Rectangle) {
	// TODO offset point
	drawScale(screen, v.automaton.water, v.automaton.waterStats, 0, 255, v.automaton.points)
}
