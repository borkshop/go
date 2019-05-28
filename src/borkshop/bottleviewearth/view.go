package bottleviewearth

import (
	"borkshop/bottle"
	"image"
	"image/color"
	"image/draw"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

var white = color.RGBA{0xff, 0xff, 0xff, 0}

func New(scale int) *View {
	rect := image.Rect(0, 0, scale, scale)
	return &View{
		rect:  rect,
		earth: image.NewAlpha(rect),
		color: image.NewRGBA(rect),
	}
}

type View struct {
	rect  image.Rectangle
	color *image.RGBA
	earth *image.Alpha
}

func (v *View) Draw(screen *anansi.Screen, rect ansi.Rectangle, gen *bottle.Generation, gpt image.Point) {
	v.draw(gen)

	grid := screen.Grid
	var pt ansi.Point
	for pt.Y = rect.Min.Y; pt.Y < rect.Max.Y; pt.Y++ {
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X++ {
			o, ok := grid.CellOffset(pt)
			if !ok {
				continue
			}
			if !pt.In(rect) {
				continue
			}
			ipt := pt.ToImage()
			ipt.Y *= 2
			r, g, b, a := v.color.At(ipt.X, ipt.Y).RGBA()
			bg := ansi.RGBA(r, g, b, a)
			grid.Attr[o] = bg.BG()
		}
	}
}

func (v *View) draw(gen *bottle.Generation) {
	rect := v.rect

	// Draw base channels directly from cellular automaton.
	var pt image.Point
	for pt.Y = rect.Min.Y; pt.Y < rect.Max.Y; pt.Y++ {
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X++ {
			cell := gen.At(pt)
			earth := uint8(gen.EarthElevationStats.Project(cell.Earth, 255))
			v.earth.SetAlpha(pt.X, pt.Y, color.Alpha{earth})
		}
	}

	draw.Draw(v.color, rect, &image.Uniform{color.Black}, image.ZP, draw.Over)
	draw.DrawMask(v.color, rect, &image.Uniform{white}, image.ZP, v.earth, image.ZP, draw.Over)
}
