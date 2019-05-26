package bottleviewtopo

import (
	"borkshop/bottle"
	"borkshop/bottleview"
	"image"
	"image/color"
	"image/draw"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

var (
	black = color.Black
	// brown = color.RGBA{155, 78, 17, 0}
	brown = color.RGBA{205, 178, 117, 0}
	blue  = color.RGBA{48, 40, 177, 0}
)

type View struct {
	rect                image.Rectangle
	water               *image.Alpha
	waterElevation      *image.Alpha
	earthElevation      *image.Alpha
	waterElevationColor *image.RGBA
	color               *image.RGBA
}

func New(scale int) *View {
	rect := image.Rect(0, 0, scale, scale)
	return &View{
		rect:                rect,
		water:               image.NewAlpha(rect),
		waterElevation:      image.NewAlpha(rect),
		earthElevation:      image.NewAlpha(rect),
		waterElevationColor: image.NewRGBA(rect),
		color:               image.NewRGBA(rect),
	}
}

var _ bottleview.View = (*View)(nil)

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
			water := uint8(gen.WaterStats.Project(cell.Water, 128))
			if water > 10 {
				water += 64
			}
			v.water.SetAlpha(pt.X, pt.Y, color.Alpha{water})
			earthElevation := uint8(gen.EarthElevationStats.Project(cell.Earth, 220) + 5)
			v.earthElevation.SetAlpha(pt.X, pt.Y, color.Alpha{earthElevation})
			waterElevation := uint8(gen.WaterElevationStats.Project(cell.Water+cell.Earth, 64) + 127)
			v.waterElevation.SetAlpha(pt.X, pt.Y, color.Alpha{waterElevation})
		}
	}

	draw.Draw(v.waterElevationColor, rect, &image.Uniform{black}, image.ZP, draw.Over)
	draw.DrawMask(v.waterElevationColor, rect, &image.Uniform{blue}, image.ZP, v.waterElevation, image.ZP, draw.Over)
	draw.Draw(v.color, rect, &image.Uniform{black}, image.ZP, draw.Over)
	draw.DrawMask(v.color, rect, &image.Uniform{brown}, image.ZP, v.earthElevation, image.ZP, draw.Over)
	draw.DrawMask(v.color, rect, v.waterElevationColor, image.ZP, v.water, image.ZP, draw.Over)
}
