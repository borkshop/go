package main

import (
	"errors"
	"flag"
	"image"
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
	return &view{
		sim: bottle.New(256),
	}
}

type view struct {
	sim *bottle.Simulation
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
	snap := v.sim.Snap()

	grid := ctx.Output.Grid
	rect := grid.Rect
	var pt ansi.Point
	for pt.Y = rect.Min.Y; pt.Y < rect.Max.Y; pt.Y++ {
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X++ {
			o, ok := grid.CellOffset(pt.Add(image.Pt(0, 1)))
			if !ok {
				continue
			}
			el := snap.At(pt.ToImage())
			value := uint8(snap.SurfaceElevationStats.Project(el.SurfaceElevation, 255))
			grid.Attr[o] = ansi.RGB(value, value, value).BG()
		}
	}

	return
}
