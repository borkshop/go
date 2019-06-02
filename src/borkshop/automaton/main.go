package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jcorbin/anansi"
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
			if err := p.Run(newApp()); platform.IsReplayDone(err) {
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

type App struct {
	ticking    int
	automaton  *Automaton
	view       View
	platesView *PlatesView
	earthView  *EarthView
	waterView  *WaterView
	mapView    *MapView
}

type View interface {
	Draw(screen *anansi.Screen, rect ansi.Rectangle)
}

func newApp() *App {
	const order = 8
	const numPlates = 5
	automaton := NewAutomaton(order, numPlates)
	platesView := NewAnansiPlatesView(automaton)
	earthView := NewAnansiEarthView(automaton)
	waterView := NewAnansiWaterView(automaton)
	mapView := NewAnansiMapView(automaton)

	// automaton.enableFaucet = true
	// automaton.disableWaterCoverage = true
	// automaton.disableSlides = true
	// automaton.disableQuakes = true
	// automaton.disablePlates = true
	// automaton.SetMountainTestPattern()

	return &App{
		automaton:  automaton,
		platesView: platesView,
		earthView:  earthView,
		waterView:  waterView,
		mapView:    mapView,
		view:       mapView,
	}
}

func (a *App) Update(ctx *platform.Context) (err error) {
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

	a.ticking += ctx.Input.CountRune('p')
	var ticks int
	if a.ticking%2 == 1 {
		ticks = 1
	}
	ticks += ctx.Input.CountRune('n')

	for i := 0; i < ticks; i++ {
		a.automaton.Tick()
	}

	switch {
	case ctx.Input.CountRune('P') > 0:
		a.view = a.platesView
	case ctx.Input.CountRune('E') > 0:
		a.view = a.earthView
	case ctx.Input.CountRune('W') > 0:
		a.view = a.waterView
	case ctx.Input.CountRune('M') > 0:
		a.view = a.mapView
	}

	// TODO thread viewport scroll offset
	a.automaton.Predraw()
	a.view.Draw(ctx.Output, ctx.Output.Grid.Rect)

	screen := ctx.Output
	screen.To(ansi.Pt(1, 1))
	screen.WriteString(fmt.Sprintf("Generation: %d\r\n", a.automaton.gen))
	screen.WriteString(fmt.Sprintf("Plate Sizes: %v\r\n", a.automaton.plateSizes))
	screen.WriteString(fmt.Sprintf("Earth Elevation: %s\r\n", a.automaton.earthStats.String()))
	screen.WriteString(fmt.Sprintf("Earthquake PID: %s\r\n", a.automaton.earthPID.String()))
	screen.WriteString(fmt.Sprintf("Water: %s\r\n", a.automaton.waterStats.String()))
	screen.WriteString(fmt.Sprintf("Water Coverage PID: %s\r\n", a.automaton.waterPID.String()))
	screen.WriteString(fmt.Sprintf("Quakes moved earth: %d\r\n", a.automaton.quake))
	screen.WriteString(fmt.Sprintf("Water flowed: %d\r\n", a.automaton.flow))

	return
}
