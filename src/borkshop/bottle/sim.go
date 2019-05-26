package bottle

import (
	"image"

	"borkshop/bottlepid"
	"borkshop/bottlestats"
	"borkshop/hilbert"
	"borkshop/xorshiftstar"
)

// Cell is a unit of the cellular automaton.
type Cell struct {
	Random xorshiftstar.Source
	Earth  int
	Water  int
}

// Grid is a slice of cells indexed by hilbert number to maximize
// memory locality of adjacent cells.
type Grid []Cell

// Generation is a snapshot of a generation.
type Generation struct {
	Scale hilbert.Scale
	Num   int
	Grid  Grid

	WaterCoverageController bottlepid.Generation

	EarthElevationStats bottlestats.Stats
	WaterElevationStats bottlestats.Stats
	WaterStats          bottlestats.Stats
	WaterCoverage       int
	EarthFlow           int
	WaterFlow           int
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
