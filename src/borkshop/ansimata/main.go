package main

import (
	"errors"
	"flag"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"borkshop/bottle"

	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
)

// Render pipeline
// Simulation.Frame
// View.Grid
// Screen.Grid

var errInt = errors.New("interrupt")

var (
	black = color.Black
	brown = color.RGBA{155, 78, 17, 0}
	blue  = color.RGBA{48, 40, 177, 0}
)

func main() {
	rand.Seed(time.Now().UnixNano()) // TODO find the right place to seed
	// TODO load config from file
	flag.Parse()
	platform.MustRun(os.Stdout, func(p *platform.Platform) error {
		for {
			if err := p.Run(newView()); platform.IsReplayDone(err) {
				continue // loop replay
			} else if err == io.EOF || err == errInt {
				return nil
			} else if err != nil {
				log.Printf("exiting due to %v", err)
				return err
			}
		}
	}, platform.FrameRate(60), platform.Config{
		LogFileName: "ansimata.log",
	})
}

func newView() *view {
	const scale = 256
	rect := image.Rect(0, 0, scale, scale)
	return &view{
		rect:                rect,
		sim:                 bottle.New(scale),
		water:               image.NewAlpha(rect),
		waterElevation:      image.NewAlpha(rect),
		earthElevation:      image.NewAlpha(rect),
		waterElevationColor: image.NewRGBA(rect),
		color:               image.NewRGBA(rect),
	}
}

type view struct {
	rect                image.Rectangle
	sim                 *bottle.Simulation
	water               *image.Alpha
	waterElevation      *image.Alpha
	earthElevation      *image.Alpha
	waterElevationColor *image.RGBA
	color               *image.RGBA
}

func (v *view) Update(ctx *platform.Context) (err error) {
	// Ctrl-C interrupts
	if ctx.Input.HasTerminal('\x03') {
		err = errInt
	}

	// Ctrl-Z suspends
	if ctx.Input.CountRune('\x1a') > 0 {
		defer func() {
			if err == nil {
				err = ctx.Suspend()
			} // else NOTE don't bother suspending, e.g. if Ctrl-C was also present
		}()
	}

	v.sim.Tick()
	gen := v.sim.Snap()
	v.draw(gen)

	grid := ctx.Output.Grid
	rect := grid.Rect
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

	return
}

func (v *view) draw(gen *bottle.Generation) {
	rect := v.rect

	// Compositor:
	// water is a full-range grayscale
	// earthElevation is a mid-range grayscale
	// waterElevation is a mid-range grayscale
	// waterElevationColor is uniform blue over uniform black with waterElevation as alpha mask
	// earthElevationColor is uniform brown over uniform black with earthElevation as alpha mask
	// color is waterElevationColor over earthElevationColor with water as alpha mask

	// Draw base channels directly from cellular automaton.
	var pt image.Point
	for pt.Y = rect.Min.Y; pt.Y < rect.Max.Y; pt.Y++ {
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X++ {
			cell := gen.At(pt)
			water := uint8(gen.WaterStats.Project(cell.Water, 255))
			v.water.SetAlpha(pt.X, pt.Y, color.Alpha{water})
			earthElevation := uint8(gen.EarthElevationStats.Project(cell.Earth, 127) + 64)
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
