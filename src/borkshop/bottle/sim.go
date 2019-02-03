package bottle

import (
	"image"

	"borkshop/hilbert"
	"borkshop/xorshiftstar"
)

const (
	quaking   = 10
	smoothing = 10
)

// Cell is a unit of the cellular automaton.
type Cell struct {
	Random           xorshiftstar.Source
	SurfaceElevation int
}

// Grid is a slice of cells indexed by hilbert number to maximize
// memory locality of adjacent cells.
type Grid []Cell

// Generation is a snapshot of a generation.
type Generation struct {
	Scale                 hilbert.Scale
	Num                   int
	Grid                  Grid
	SurfaceElevationStats Stats
}

func newGeneration(scale hilbert.Scale) *Generation {
	return &Generation{
		Scale: scale,
		Grid:  make(Grid, scale*scale),
	}
}

// At returns a pointer to the cell at a given point, for any point.
func (gen *Generation) At(pt image.Point) *Cell {
	return &gen.Grid[gen.Scale.Encode(pt)]
}

// Simulation is a pre-allocated set of generations for the
// simulation.
type Simulation struct {
	prev, next, snap *Generation
	scale            hilbert.Scale
}

// New creates a new simulation with pre-allocated generations for a
// given scale.
func New(scale int) *Simulation {
	sim := &Simulation{scale: hilbert.Scale(scale)}
	sim.next = newGeneration(sim.scale)
	sim.prev = newGeneration(sim.scale)
	sim.reset(sim.prev)
	return sim
}

// Tick runs a tick of the sim simulation.
func (sim *Simulation) Tick() {
	sim.tick(sim.next, sim.prev)
	sim.next, sim.prev = sim.prev, sim.next
}

// Snap returns the most recent generation.
func (sim *Simulation) Snap() *Generation {
	return sim.prev
}

func (sim *Simulation) reset(gen *Generation) {
	var pt image.Point
	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			hilbertNumber := sim.scale.Encode(pt)
			cell := &gen.Grid[hilbertNumber]
			cell.Random.Seed(int64(hilbertNumber + 1))
			cell.SurfaceElevation = 0
		}
	}
}

func (sim *Simulation) tick(next, prev *Generation) {
	next.SurfaceElevationStats.Reset()

	// Next generation.
	next.Num = prev.Num + 1

	var pt image.Point

	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			hilbertNumber := sim.scale.Encode(pt)
			nc := &next.Grid[hilbertNumber]
			pc := &prev.Grid[hilbertNumber]
			nc.Random = pc.Random
			nc.SurfaceElevation = pc.SurfaceElevation
		}
	}

	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			a := &next.Grid[sim.scale.Encode(pt)]
			b := &next.Grid[sim.scale.Encode(pt.Add(image.Pt(1, 0)))]
			c := &next.Grid[sim.scale.Encode(pt.Add(image.Pt(0, 1)))]

			// Quaking
			if a.Random.Uint64()&1 == 0 {
				a.SurfaceElevation += quaking
				b.SurfaceElevation -= quaking
			}
			if a.Random.Uint64()&1 == 0 {
				a.SurfaceElevation += quaking
				c.SurfaceElevation += quaking
			}

			// Angle of repose
			if a.SurfaceElevation > b.SurfaceElevation+10 {
				diff := (a.SurfaceElevation - b.SurfaceElevation) / smoothing
				a.SurfaceElevation -= diff
				b.SurfaceElevation += diff
			}

			if a.SurfaceElevation > c.SurfaceElevation+10 {
				diff := (a.SurfaceElevation - c.SurfaceElevation) / smoothing
				a.SurfaceElevation -= diff
				c.SurfaceElevation += diff
			}
		}
	}

	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			a := &next.Grid[sim.scale.Encode(pt)]
			next.SurfaceElevationStats.Add(a.SurfaceElevation)
		}
	}
}
