package bottle

import (
	"image"

	"borkshop/bottlepid"
	"borkshop/bottlestats"
	"borkshop/hilbert"
	"borkshop/xorshiftstar"
)

// NumPlates is the number of tectonic plate classes, by shared direction of
// movement.
const NumPlates = 5

// Cell is a unit of the cellular automaton.
type Cell struct {
	Random xorshiftstar.Source
	Earth  int
	Water  int
	Plate  uint8
}

// Grid is a slice of cells indexed by hilbert number to maximize
// memory locality of adjacent cells.
type Grid []Cell

// Generation is a snapshot of a generation.
type Generation struct {
	Scale hilbert.Scale
	Num   int
	Grid  Grid

	EarthElevationStats bottlestats.Stats
	WaterElevationStats bottlestats.Stats
	WaterStats          bottlestats.Stats
	EarthFlow           int
	WaterFlow           int
	QuakeFlow           int
	PlateSizes          [NumPlates]int

	WaterCoverage           int
	WaterCoverageController bottlepid.Generation
}

// NewGeneration constructs a generation at a particular scale.
//
// Simulations treat generations as frame buffers, alternating and recycling
// the previous generation as the next generation's memory.
func NewGeneration(scale hilbert.Scale) *Generation {
	area := int(scale * scale)
	return &Generation{
		Scale: scale,
		Grid:  make(Grid, area),
	}
}

// At returns a pointer to the cell at a given point, for any point.
func (gen *Generation) At(pt image.Point) *Cell {
	return &gen.Grid[gen.Scale.Encode(pt)]
}
