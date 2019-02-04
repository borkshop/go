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
		for pt.X = rect.Min.X; pt.X < rect.Max.X; pt.X += 2 {
			// Simulation
			//   NW NN NE
			//   WW CC EE EX
			//   SW SS SE
			// Render
			//   -- -- --
			//   -- O  P
			//   -- -- --

			o, ok := grid.CellOffset(pt)
			if !ok {
				continue
			}
			p, pk := grid.CellOffset(pt.Add(image.Pt(1, 0)))

			// Simulation point.
			spt := pt
			spt.Y *= 2

			nw := snap.At(spt.Add(image.Pt(0, 0)).ToImage())
			nn := snap.At(spt.Add(image.Pt(0, 1)).ToImage())
			sw := snap.At(spt.Add(image.Pt(0, 2)).ToImage())
			ww := snap.At(spt.Add(image.Pt(1, 0)).ToImage())
			cc := snap.At(spt.Add(image.Pt(1, 1)).ToImage())
			ee := snap.At(spt.Add(image.Pt(1, 2)).ToImage())
			ne := snap.At(spt.Add(image.Pt(2, 0)).ToImage())
			ss := snap.At(spt.Add(image.Pt(2, 1)).ToImage())
			se := snap.At(spt.Add(image.Pt(2, 2)).ToImage())
			ex := snap.At(spt.Add(image.Pt(1, 3)).ToImage())

			nb := uint8(snap.WaterStats.Project(nn.Water, 4) << 6)
			sb := uint8(snap.WaterStats.Project(ss.Water, 4) << 4)
			eb := uint8(snap.WaterStats.Project(ee.Water, 4) << 2)
			wb := uint8(snap.WaterStats.Project(ww.Water, 4) << 0)
			obyte := nb | sb | eb | wb
			orune := lineArtRunes[lineArtRuneOffsets[obyte]]

			oavr := (nw.Earth + sw.Earth + ne.Earth + se.Earth) / 4
			oearth := uint8(snap.EarthElevationStats.Project(oavr, 127))

			ofg := ansi.RGB(0, 0, 0xff)
			obg := ansi.RGB(oearth+63, oearth*2/3+63, oearth/3+63)

			grid.Rune[o] = orune
			grid.Attr[o] = obg.BG() | ofg.FG()

			if !pk {
				continue
			}

			pw := uint8(snap.WaterStats.Project((cc.Water+ee.Water)/2, 4) << 2)
			pe := uint8(snap.WaterStats.Project((ee.Water+ex.Water)/2, 4) << 0)
			pb := pe | pw
			pr := lineArtRunes[lineArtRuneOffsets[pb]]

			pf := (ne.Earth + se.Earth) / 2
			pv := uint8(snap.EarthElevationStats.Project(pf, 127))

			pfg := ansi.RGB(0, 0, 0xff)
			pbg := ansi.RGB(pv+63, pv*2/3+63, pv/3+63)

			grid.Rune[p] = pr
			grid.Attr[p] = pbg.BG() | pfg.FG()
		}
	}

	return
}
