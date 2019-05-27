package bottleviewplate

import (
	"borkshop/bottle"
	"image"
	"image/color"

	"github.com/hsluv/hsluv-go"
	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

var colors [bottle.NumPlates]color.RGBA

func newColor(over, under int) color.RGBA {
	r, g, b := hsluv.HsluvToRGB(
		360*float64(over)/float64(under),
		75,
		50,
	)
	return color.RGBA{
		uint8(r * 0xff),
		uint8(g * 0xff),
		uint8(b * 0xff),
		0xff,
	}
}

func init() {
	for i := 0; i < bottle.NumPlates; i++ {
		colors[i] = newColor(i, bottle.NumPlates)
	}
}

func New(scale int) *View {
	rect := image.Rect(0, 0, scale, scale)
	return &View{
		rect:  rect,
		color: image.NewRGBA(rect),
	}
}

type View struct {
	rect  image.Rectangle
	color *image.RGBA
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
			v.color.SetRGBA(pt.X, pt.Y, colors[cell.Plate])
		}
	}
}
