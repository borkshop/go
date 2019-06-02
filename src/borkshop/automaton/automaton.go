package main

import (
	"borkshop/stencil"
	"image"
)

type Automaton struct {
	gen       int
	order     int
	length    int
	area      int
	rect      image.Rectangle
	numPlates int

	stencil3 [][2]int
	stencil5 [][4]int
	stencil9 [][8]int
	points   []image.Point
	temp3s   [][3]int64

	entropy []int64

	// Tectonic plates
	plates        []int64
	plate5s       [][5]int64
	plateSizes    []int64
	plateWeights  []int64
	disablePlates bool

	// Earth quakes
	earth         []int64
	earthStats    Stats64
	earth3s       [][3]int64
	quakeVectors  []image.Point
	quake         int64
	earthPID      PID
	disableQuakes bool

	// Earth slides
	repose        []int64
	slide         int64
	disableSlides bool

	// Water coverage control
	water                 []int64
	water3s               [][3]int64
	waterStats            Stats64
	waterCoverage         int64
	waterPID              PID
	disableWaterCoverage  bool
	enableFaucet          bool
	enableDrain           bool
	waterAdjustmentVolume int64
	significantWater      int64

	// Watershed
	flow             int64
	disableWatershed bool
}

func NewAutomaton(order int, numPlates int) *Automaton {
	length := 1 << uint(order)
	area := 1 << uint(order*2)
	rect := image.Rect(0, 0, length, length)

	stencil3 := make([][2]int, area)
	stencil5 := make([][4]int, area)
	stencil9 := make([][8]int, area)
	temp3s := make([][3]int64, area)
	points := make([]image.Point, area)
	entropy := make([]int64, area)
	plates := make([]int64, area)
	plate5s := make([][5]int64, area)
	plateSizes := make([]int64, numPlates)
	plateWeights := make([]int64, numPlates)
	earth := make([]int64, area)
	earth3s := make([][3]int64, area)
	quakeVectors := make([]image.Point, numPlates)
	repose := make([]int64, area)
	water := make([]int64, area)
	water3s := make([][3]int64, area)

	stencil.WriteHilbertPoints(points, length)
	stencil.WriteHilbertStencil3Table(stencil3, length)
	stencil.WriteHilbertStencil5Table(stencil5, length)
	stencil.WriteHilbertStencil9Table(stencil9, length)
	stencil.InitInt64Vector(repose, 0x1f)

	earthPID := PID{
		Target:           0xff,
		ProportionalGain: 0xfff,
		IntegralGain:     0xff,
		DifferentialGain: 0xff,
		Min:              0x0,
		Max:              0xffffff,
	}

	waterPID := PID{
		Target:           int64(area) / 2,
		ProportionalGain: 0xfff,
		IntegralGain:     0xff,
		DifferentialGain: 0x1,
		Min:              -0xffffffff,
		Max:              0xffffffff,
	}

	waterAdjustmentVolume := int64(0xf)
	significantWater := int64(0xf)

	stencil.WriteSequenceInt64Vector(entropy)
	WriteNextRandomInt64Vector(entropy)

	WriteQuakeVectors(quakeVectors)

	WriteRandomPlateVector(plates, entropy, numPlates)
	MeasurePlateSizes(plateSizes, plates)

	automaton := &Automaton{
		order:     order,
		length:    length,
		area:      area,
		rect:      rect,
		numPlates: numPlates,

		stencil3: stencil3,
		stencil5: stencil5,
		stencil9: stencil9,
		temp3s:   temp3s,
		points:   points,

		entropy: entropy,

		plates:       plates,
		plate5s:      plate5s,
		plateSizes:   plateSizes,
		plateWeights: plateWeights,

		earth:        earth,
		earth3s:      earth3s,
		quakeVectors: quakeVectors,
		earthPID:     earthPID,
		repose:       repose,

		water:                 water,
		water3s:               water3s,
		waterPID:              waterPID,
		waterAdjustmentVolume: waterAdjustmentVolume,
		significantWater:      significantWater,
	}

	return automaton
}

func (a *Automaton) Predraw() {
	WriteStatsFromInt64Vector(&a.earthStats, a.earth)
	WriteStatsFromInt64Vector(&a.waterStats, a.water)
}

func (a *Automaton) Tick() {
	// Plates
	if !a.disablePlates {
		stencil.WriteStencil5Int64Vector(a.plate5s, a.plates, a.stencil5)
		WriteNextPlateVector(a.plates, a.plate5s, a.entropy, a.plateSizes, a.plateWeights)
		MeasurePlateSizes(a.plateSizes, a.plates)
		WriteNextRandomInt64Vector(a.entropy)
	}

	// Quakes
	if !a.disableQuakes {
		WriteStatsFromInt64Vector(&a.earthStats, a.earth)
		a.earthPID.Tick(a.earthStats.Spread())
		stencil.WriteStencil3Int64Vector(a.earth3s, a.earth, a.stencil3)
		Quake(a.temp3s, &a.quake, a.earth3s, a.plates, a.quakeVectors, a.earthPID.Control, a.earthPID.Max, a.entropy)
		stencil.EraseInt64Vector(a.earth)
		stencil.AddInt64VectorStencil3(a.earth, a.temp3s, a.stencil3)
	}

	// Slides
	if !a.disableSlides {
		stencil.WriteStencil3Int64Vector(a.earth3s, a.earth, a.stencil3)
		SlideInt64Vector(a.temp3s, &a.slide, a.earth3s, a.repose, a.entropy)
		stencil.EraseInt64Vector(a.earth)
		stencil.AddInt64VectorStencil3(a.earth, a.temp3s, a.stencil3)
		WriteNextRandomInt64Vector(a.entropy)
	}

	// Watershed
	if !a.disableWatershed {
		stencil.WriteStencil3Int64Vector(a.earth3s, a.earth, a.stencil3)
		a.flow = 0

		// Lateral
		stencil.WriteStencil3Int64Vector(a.water3s, a.water, a.stencil3)
		WatershedInt64Vector(a.temp3s, &a.flow, a.water3s, a.earth3s, a.entropy, 1)
		stencil.EraseInt64Vector(a.water)
		stencil.AddInt64VectorStencil3(a.water, a.temp3s, a.stencil3)
		WriteNextRandomInt64Vector(a.entropy)

		// Longitudinal
		stencil.WriteStencil3Int64Vector(a.water3s, a.water, a.stencil3)
		WatershedInt64Vector(a.temp3s, &a.flow, a.water3s, a.earth3s, a.entropy, 2)
		stencil.EraseInt64Vector(a.water)
		stencil.AddInt64VectorStencil3(a.water, a.temp3s, a.stencil3)
		WriteNextRandomInt64Vector(a.entropy)
	}

	// Water Coverage
	if !a.disableWaterCoverage {
		WriteStatsFromInt64Vector(&a.waterStats, a.water)
		MeasureWaterCoverage(&a.waterCoverage, a.water, a.significantWater)
		a.waterPID.Tick(a.waterCoverage)
		AdjustWaterInt64Vector(a.water, a.waterPID.Control, a.entropy, a.waterAdjustmentVolume)
		WriteNextRandomInt64Vector(a.entropy)
	}

	// Water faucet and drain test
	if a.enableFaucet {
		a.water[a.area/2] = 10
	}
	if a.enableDrain {
		a.water[0] = 0
	}

	a.gen++
}

func (a *Automaton) SetTowerTestPattern() {
	stencil.InitInt64Vector(a.earth, 0)
	a.earth[a.area/2] = 1000
}

func (a *Automaton) SetMountainTestPattern() {
	for i := 0; i < a.area; i++ {
		pt := a.points[i]
		z := int64(a.length)
		x := mag64(int64(pt.X) - z/2)
		y := mag64(int64(pt.Y) - z/2)
		a.earth[i] = z - x - y
	}
}

func (a *Automaton) SetHilbertMountainTestPattern() {
	for i := 0; i < a.area; i++ {
		a.earth[i] = int64(i)
	}
}