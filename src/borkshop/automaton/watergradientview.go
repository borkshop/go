package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hsluv/hsluv-go"
)

type WaterGradientView struct {
	automaton  *Automaton
	rect       image.Rectangle
	hue        *image.RGBA
	mask       *image.RGBA
	angles     []uint8
	magnitudes []int64
	stats      Stats64
}

func NewWaterGradientView(automaton *Automaton) *WaterGradientView {
	rect := image.Rectangle{image.ZP, image.Pt(automaton.length, automaton.length)}
	return &WaterGradientView{
		automaton:  automaton,
		rect:       rect,
		hue:        image.NewRGBA(rect),
		mask:       image.NewRGBA(rect),
		angles:     make([]uint8, automaton.area),
		magnitudes: make([]int64, automaton.area),
	}
}

func (v *WaterGradientView) Name() string {
	return "water-gradient"
}

func (v *WaterGradientView) Draw(screen *image.RGBA, rect image.Rectangle) {
	measureGradient64FromStencil3(v.angles, v.magnitudes, v.automaton.waterGradient3s)
	WriteStatsFromInt64Vector(&v.stats, v.magnitudes)
	drawAngles(v.hue, v.angles, v.automaton.points)
	drawMagnitudes(v.mask, v.magnitudes, &v.stats, v.automaton.points)
	draw.Draw(screen, v.rect, &image.Uniform{color.White}, image.ZP, draw.Src)
	// draw.Draw(screen, v.rect, v.hue, image.ZP, draw.Over)
	draw.DrawMask(screen, v.rect, v.hue, image.ZP, v.mask, image.ZP, draw.Over)
}

func measureGradient64FromStencil3(angles []uint8, magnitudes []int64, src [][3]int64) {
	for i := 0; i < len(src); i++ {
		x := float64(src[i][1])
		y := float64(src[i][2])
		angles[i] = uint8(math.Atan2(y, x) * 128 / math.Pi)
		magnitudes[i] = int64(math.Sqrt(float64(x*x + y*y)))
	}
}

func drawAngles(dst *image.RGBA, angles []uint8, points []image.Point) {
	for i := 0; i < len(points); i++ {
		c := newVectorColor(angles[i])
		pt := points[i]
		dst.SetRGBA(pt.X, pt.Y, c)
	}
}

func drawMagnitudes(dst *image.RGBA, magnitudes []int64, stats *Stats64, points []image.Point) {
	for i := 0; i < len(points); i++ {
		pt := points[i]
		dst.Set(pt.X, pt.Y, color.Alpha{
			A: uint8(stats.Project(magnitudes[i], 255)),
		})
	}
}

func newVectorColor(angle uint8) color.RGBA {
	r, g, b := hsluv.HsluvToRGB(
		360*float64(angle)/256,
		100,
		30,
	)
	return color.RGBA{
		uint8(r * 0xff),
		uint8(g * 0xff),
		uint8(b * 0xff),
		0xff,
	}
}
