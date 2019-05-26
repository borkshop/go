package main

import (
	"borkshop/bottle"
	"borkshop/bottlemudslide"
	"borkshop/bottlepid"
	"borkshop/bottlesimstats"
	"borkshop/bottletoposimplex"
	"borkshop/bottleview"
	"borkshop/bottleviewtopo"
	"borkshop/bottlewatercoverage"
	"borkshop/bottlewatershed"
	"borkshop/hilbert"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
)

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
	const scale = 256
	rect := image.Rect(0, 0, scale, scale)
	mudslide := &bottlemudslide.Simulation{
		Scale:  hilbert.Scale(scale),
		Repose: 2,
	}
	watershed := &bottlewatershed.Simulation{
		Scale: hilbert.Scale(scale),
	}
	waterCoverage := &bottlewatercoverage.Simulation{
		Controller: bottlepid.Controller{
			Proportional: bottlepid.G(0xff, 1),
			Integral:     bottlepid.G(1, 1),
			Differential: bottlepid.G(1, 1),
			Value:        scale * scale / 3,
			Min:          -0xffffffff,
			Max:          0xffffffff,
		},
	}
	res := bottle.Resetters{
		bottletoposimplex.New(scale),
		// bottleflood.New(scale, 0),
	}
	next := bottle.NewGeneration(scale)
	prev := bottle.NewGeneration(scale)
	res.Reset(prev)
	ticker := bottle.Tickers{
		bottlesimstats.Pre{},
		mudslide,
		watershed,
		waterCoverage,
		bottlesimstats.Post{},
	}
	topo := bottleviewtopo.New(scale)
	return &view{
		rect:          rect,
		ticker:        ticker,
		resetter:      res,
		waterCoverage: waterCoverage,
		prev:          prev,
		next:          next,
		view:          topo,
	}
}

type view struct {
	rect          image.Rectangle
	ticker        bottle.Ticker
	resetter      bottle.Resetter
	waterCoverage *bottlewatercoverage.Simulation
	view          bottleview.View
	next, prev    *bottle.Generation
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

	for i := 0; i < 1; i++ {
		v.ticker.Tick(v.next, v.prev)
		v.next, v.prev = v.prev, v.next
	}
	v.view.Draw(ctx.Output, ctx.Output.Grid.Rect, v.next, image.ZP)

	gen := v.next
	screen := ctx.Output
	screen.To(ansi.Pt(1, 1))
	screen.WriteString(fmt.Sprintf("EarthElevation %d...%d\r\n", gen.EarthElevationStats.Min, gen.EarthElevationStats.Max))
	screen.WriteString(fmt.Sprintf("WaterElevation %d...%d\r\n", gen.WaterElevationStats.Min, gen.WaterElevationStats.Max))
	screen.WriteString(fmt.Sprintf("Water %d...%d\r\n", gen.WaterStats.Min, gen.WaterStats.Max))
	screen.WriteString(fmt.Sprintf("WaterCoverage %d\r\n", gen.WaterCoverage))
	screen.WriteString(fmt.Sprintf("     Converge %d\r\n", v.waterCoverage.Controller.Value))
	screen.WriteString(fmt.Sprintf(" C %d\r\n", gen.WaterCoverageController.Proportional))
	screen.WriteString(fmt.Sprintf(" P %d\r\n", gen.WaterCoverageController.Integral))
	screen.WriteString(fmt.Sprintf(" I %d\r\n", gen.WaterCoverageController.Differential))
	screen.WriteString(fmt.Sprintf(" D %d\r\n", gen.WaterCoverageController.Control))
	screen.WriteString(fmt.Sprintf("WaterFlow %d\r\n", gen.WaterFlow))
	screen.WriteString(fmt.Sprintf("EarthFlow %d\r\n", gen.EarthFlow))

	return
}
