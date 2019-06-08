// +build js

package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"syscall/js"
	"time"

	"borkshop/stats"
)

var errInt = errors.New("interrupt")

func main() {
	// TODO stop using the global locked rand
	rand.Seed(time.Now().UnixNano()) // TODO find the right place to seed
	var ctx imContext
	if err := ctx.Run(newApp()); err != nil {
		log.Fatalln(err)
	}
}

type App struct {
	ticking     bool
	ticks       chan chan struct{}
	tickPending chan struct{}
	tickTimes   stats.Times

	automaton  *Automaton
	view       View
	platesView *PlatesView
	earthView  *EarthView
	waterView  *WaterView
	mapView    *MapView
}

type View interface {
	Draw(screen *image.RGBA, rect image.Rectangle)
}

func newApp() *App {
	const order = 8
	const numPlates = 5
	automaton := NewAutomaton(order, numPlates)
	platesView := NewPlatesView(automaton)
	earthView := NewEarthView(automaton)
	waterView := NewWaterView(automaton)
	mapView := NewMapView(automaton)

	// automaton.enableFaucet = true
	// automaton.disableWaterCoverage = true
	// automaton.disableSlides = true
	// automaton.disableQuakes = true
	// automaton.disablePlates = true
	// automaton.SetMountainTestPattern()

	return &App{
		tickTimes:  stats.MakeTimes(120),
		automaton:  automaton,
		platesView: platesView,
		earthView:  earthView,
		waterView:  waterView,
		mapView:    mapView,
		view:       mapView,
	}
}

func (a *App) Open() (io.Closer, error) {
	a.ticks = make(chan chan struct{}, 1)
	go a.ticker()
	return a, nil
}

func (a *App) Close() error {
	close(a.ticks)
	return nil
}

func (a *App) ticker() {
	for req := range a.ticks {
		a.automaton.Tick()
		a.automaton.Predraw()
		close(req)
	}
}

func (a *App) Update(ctx *imContext) (err error) {
	var tick bool
	if ctx.key.press == 'p' {
		a.ticking = !a.ticking
		ctx.animating = a.ticking
	}

	if a.ticking {
		// while playing, tick per animation update
		tick = ctx.elapsed > 0
	} else if ctx.key.press == 'n' {
		// when paused, allow manual stepping
		tick = true
	}

	draw := tick

	switch ctx.key.press {
	case 'P':
		a.view = a.platesView
		draw = true
	case 'E':
		a.view = a.earthView
		draw = true
	case 'W':
		a.view = a.waterView
		draw = true
	case 'M':
		a.view = a.mapView
		draw = true
	}

	if a.tickPending != nil {
		select {
		case <-a.tickPending:
			a.tickPending = nil
			draw = true
		default:
			// not done yet
			draw = false
		}
	} else if tick {
		a.tickTimes.Collect(ctx.now)
		a.tickPending = make(chan struct{}, 1)
		a.ticks <- a.tickPending
		draw = false
	}

	if draw {
		// TODO thread viewport scroll offset
		ctx.clearScreen()
		a.view.Draw(ctx.screen, ctx.screen.Rect)

		ctx.clearInfo()
		ctx.infof("Generation: %d\r\n", a.automaton.gen)
		ctx.infof("Plate Sizes: %v\r\n", a.automaton.plateSizes)
		ctx.infof("Earth Elevation: %s\r\n", a.automaton.earthStats.String())
		ctx.infof("Earthquake PID: %s\r\n", a.automaton.earthPID.String())
		ctx.infof("Quakes moved earth: %d\r\n", a.automaton.quake)
		ctx.infof("Slides moved earth: %d\r\n", a.automaton.slide)
		ctx.infof("Water: %s\r\n", a.automaton.waterStats.String())
		ctx.infof("Water Coverage PID: %s\r\n", a.automaton.waterPID.String())
		ctx.infof("Water created or destroyed: %d\r\n", a.automaton.waterAdjusted)
		ctx.infof("Water flowed: %d\r\n", a.automaton.flow)
	}

	if ctx.profTiming {
		ctx.proff("%v TPS\n", a.tickTimes.CountRecent(ctx.now, time.Second))
	}

	return
}

var (
	fetch      = js.Global().Get("fetch")
	consoleLog = js.Global().Get("console")
)

func uploadBytes(name string, b []byte) {
	uploadURL := os.Getenv("upload")

	if uploadURL == "" {
		consoleLog.Get("log").Invoke(
			name,
			js.Global().Call("btoa", js.TypedArrayOf(b)),
		)
		return
	}

	if err := postBytes(
		fmt.Sprintf("%s?name=%s", uploadURL, url.QueryEscape(name)),
		"application/octet-stream",
		b,
	); err != nil {
		log.Printf("upload %v failed: %v", name, err)
	} else {
		log.Printf("uploaded %v", name)
	}
}

func postBytes(url, contentType string, b []byte) error {
	res, err := await(fetch.Invoke(url, map[string]interface{}{
		"method":   "POST",
		"redirect": "error",
		"headers": map[string]interface{}{
			"Content-Type": contentType,
		},
		"body": js.TypedArrayOf(b),
	}))
	if err != nil {
		return err
	}
	if !res.Get("ok").Bool() {
		return fmt.Errorf("%v %v", res.Get("status").Int(), res.Get("statusText").String())
	}
	return nil
}

func await(promise js.Value) (js.Value, error) {
	done := make(chan js.Value)
	fail := make(chan error)
	promise.Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			done <- args[0]
			return nil
		}),
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fail <- errors.New(args[0].String())
			return nil
		}),
	)
	select {
	case val := <-done:
		return val, nil
	case err := <-fail:
		return js.Undefined(), err
	}
}
