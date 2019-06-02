// +build js

package main

import (
	"errors"
	"image"
	"log"
	"math/rand"
	"time"
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
	ticking    int
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
		automaton:  automaton,
		platesView: platesView,
		earthView:  earthView,
		waterView:  waterView,
		mapView:    mapView,
		view:       mapView,
	}
}

func (a *App) Update(ctx *imContext) (err error) {
	if ctx.key.press == 'p' {
		a.ticking++
	}
	var ticks int
	if a.ticking%2 == 1 {
		ticks = 1
	}
	if ctx.key.press == 'n' {
		ticks++
	}

	for i := 0; i < ticks; i++ {
		a.automaton.Tick()
	}

	switch ctx.key.press {
	case 'P':
		a.view = a.platesView
	case 'E':
		a.view = a.earthView
	case 'W':
		a.view = a.waterView
	case 'M':
		a.view = a.mapView
	}

	// TODO thread viewport scroll offset
	a.automaton.Predraw()
	a.view.Draw(ctx.screen, ctx.screen.Rect)

	ctx.infof("Generation: %d\r\n", a.automaton.gen)
	ctx.infof("Plate Sizes: %v\r\n", a.automaton.plateSizes)
	ctx.infof("Earth Elevation: %s\r\n", a.automaton.earthStats.String())
	ctx.infof("Earthquake PID: %s\r\n", a.automaton.earthPID.String())
	ctx.infof("Water: %s\r\n", a.automaton.waterStats.String())
	ctx.infof("Water Coverage PID: %s\r\n", a.automaton.waterPID.String())
	ctx.infof("Quakes moved earth: %d\r\n", a.automaton.quake)
	ctx.infof("Slides moved earth: %d\r\n", a.automaton.slide)
	ctx.infof("Water flowed: %d\r\n", a.automaton.flow)

	return
}
