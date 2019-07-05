// +build js

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"runtime/pprof"
	"runtime/trace"
	"syscall/js"
	"time"

	"borkshop/stats"
)

var errInt = errors.New("interrupt")

var perfFlagsPattern = regexp.MustCompile(`\b(trace)|(cpuProfile)\b`)

type perfFlags uint8

const (
	tracePerfFlag perfFlags = 1 << iota
	cpuProfilePerfFlag
)

func parsePerfFlags() (flags perfFlags) {
	hash := js.Global().Get("location").Get("hash").String()
	for _, match := range perfFlagsPattern.FindAllStringSubmatch(hash, -1) {
		if match[1] != "" {
			flags |= tracePerfFlag
		}
		if match[2] != "" {
			flags |= cpuProfilePerfFlag
		}
	}
	return flags
}

func main() {
	flags := parsePerfFlags()
	fn := run
	if flags&tracePerfFlag != 0 {
		fn = withTrace(fn)
	}
	if flags&cpuProfilePerfFlag != 0 {
		fn = withCPUProfile(fn)
	}
	if err := fn(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	// TODO stop using the global locked rand
	rand.Seed(time.Now().UnixNano()) // TODO find the right place to seed
	var ctx imContext
	return ctx.Run(newApp())
}

type App struct {
	ticking     bool
	ticks       chan chan struct{}
	tickPending chan struct{}
	tickTimes   stats.Times

	automaton         *Automaton
	view              View
	mapView           *MapView
	platesView        *PlatesView
	earthView         *EarthView
	waterView         *WaterView
	waterGradientView *WaterGradientView
}

type View interface {
	Draw(screen *image.RGBA, rect image.Rectangle)
	Name() string
}

func newApp() *App {
	const order = 8
	const numPlates = 5
	automaton := NewAutomaton(order, numPlates)
	mapView := NewMapView(automaton)
	platesView := NewPlatesView(automaton)
	earthView := NewEarthView(automaton)
	waterView := NewWaterView(automaton)
	waterGradientView := NewWaterGradientView(automaton)

	return &App{
		tickTimes:         stats.MakeTimes(120),
		automaton:         automaton,
		view:              mapView,
		mapView:           mapView,
		platesView:        platesView,
		earthView:         earthView,
		waterView:         waterView,
		waterGradientView: waterGradientView,
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
	case '1':
		a.view = a.mapView
		draw = true
	case '2':
		a.view = a.platesView
		draw = true
	case '3':
		a.view = a.earthView
		draw = true
	case '4':
		a.view = a.waterView
		draw = true
	case '5':
		a.view = a.waterGradientView
		draw = true

	case 'r':
		a.automaton.Reset()
		draw = true
	case 'm':
		a.automaton.SetMountainTestPattern()
		draw = true
	case 't':
		a.automaton.SetTowerTestPattern()
		draw = true
	case 'h':
		a.automaton.SetHilbertMountainTestPattern()
		draw = true

	case 'f':
		a.automaton.enableFaucet = true
		draw = true
	case 'F':
		a.automaton.enableFaucet = false
		draw = true
	case 'd':
		a.automaton.enableDrain = true
		draw = true
	case 'D':
		a.automaton.enableDrain = false
		draw = true
	case 'c':
		a.automaton.disableWaterCoverage = false
		draw = true
	case 'C':
		a.automaton.disableWaterCoverage = true
		draw = true
	case 'w':
		a.automaton.disableWatershed = false
		draw = true
	case 'W':
		a.automaton.disableWatershed = true
		draw = true
	case 's':
		a.automaton.disableSlides = false
		draw = true
	case 'S':
		a.automaton.disableSlides = true
		draw = true
	case 'q':
		a.automaton.disableQuakes = false
		draw = true
	case 'Q':
		a.automaton.disableQuakes = true
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
		ctx.infof("View:                       %s [1-5]\n", a.view.Name())
		ctx.infof("Generation:                 %d\n", a.automaton.gen)
		ctx.infof("Plate Sizes:                %v\n", a.automaton.plateSizes)
		ctx.infof("Earth Elevation:            %s\n", a.automaton.earthStats.String())
		ctx.infof("Water:                      %s\n", a.automaton.waterStats.String())
		ctx.infof("Earthquake PID:             %s\n", a.automaton.earthPID.String())
		ctx.infof("Precipitation PID:          %s\n", a.automaton.precipitationPID.String())
		ctx.infof("Quakes moved earth:         %d\n", a.automaton.totalQuake)
		ctx.infof("Slides moved earth:         %d\n", a.automaton.totalSlide)
		ctx.infof("Earth eroded:               %d\n", a.automaton.totalErosion)
		ctx.infof("Precipitation:              %d\n", a.automaton.totalPrecipitation)
		ctx.infof("Evaporation:                %d\n", a.automaton.totalEvaporation)
		ctx.infof("Water shed:                 %d\n", a.automaton.totalWatershed)
		ctx.infof("[f/F]aucet open:            %v\n", a.automaton.enableFaucet)
		ctx.infof("[d/D]rain open:             %v\n", a.automaton.enableDrain)
		ctx.infof("Water [c/C]overage running: %v\n", !a.automaton.disableWaterCoverage)
		ctx.infof("[w/W]atershed running:      %v\n", !a.automaton.disableWatershed)
		ctx.infof("Mud[s/S]lides running:      %v\n", !a.automaton.disableSlides)
		ctx.infof("[q/Q]uakes running:         %v\n", !a.automaton.disableQuakes)

	}

	if ctx.profTiming {
		ctx.proff("%v TPS\n", a.tickTimes.CountRecent(ctx.now, time.Second))
	}

	return
}

func withTrace(f func() error) func() error {
	return func() error {
		log.Printf("enabling execution tracing")
		var buf bytes.Buffer
		buf.Grow(1024 * 1024)
		err := trace.Start(&buf)
		if err == nil {
			err = f()
			trace.Stop()
			uploadBytes("trace.out", buf.Bytes())
		}
		return err
	}
}

func withCPUProfile(f func() error) func() error {
	return func() error {
		log.Printf("enabling cpu profiling")
		var buf bytes.Buffer
		buf.Grow(32 * 1024)
		err := pprof.StartCPUProfile(&buf)
		if err == nil {
			err = f()
			pprof.StopCPUProfile()
			uploadBytes("prof.cpu", buf.Bytes())
		}
		return err
	}
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
		fmt.Sprintf("%s?name=XXX/%s", uploadURL, url.QueryEscape(name)),
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
