package bottle

import (
	"image"
	"image/draw"

	"borkshop/hilbert"
	"borkshop/xorshiftstar"
)

var x = draw.Draw

const (
	quaking   = 1
	smoothing = 10
	repose    = 2
	flowing   = 10
	flood     = 5
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

	EarthElevationStats Stats
	WaterElevationStats Stats
	WaterStats          Stats
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
			cel := &gen.Grid[hilbertNumber]
			cel.Random.Seed(int64(hilbertNumber + 1))
			cel.Earth = 0
			cel.Water = flood
		}
	}
}

func (sim *Simulation) tick(next, prev *Generation) {
	next.EarthElevationStats.Reset()
	next.WaterElevationStats.Reset()

	// Next generation.
	next.Num = prev.Num + 1

	var pt image.Point

	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			hilbertNumber := sim.scale.Encode(pt)
			nc := &next.Grid[hilbertNumber]
			pc := &prev.Grid[hilbertNumber]
			nc.Random = pc.Random
			nc.Earth = pc.Earth
			nc.Water = pc.Water
		}
	}

	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			cel := &next.Grid[sim.scale.Encode(pt)]
			lat := &next.Grid[sim.scale.Encode(pt.Add(image.Pt(1, 0)))]
			lon := &next.Grid[sim.scale.Encode(pt.Add(image.Pt(0, 1)))]

			// Quaking
			{
				if cel.Random.Uint64()&1 == 0 {
					cel.Earth += quaking
					lat.Earth -= quaking
				}
				if cel.Random.Uint64()&1 == 0 {
					cel.Earth += quaking
					lon.Earth += quaking
				}
			}

			// Smoothing (feaux-erosion)
			{
				latdel := (cel.Earth - lat.Earth) / smoothing
				londel := (cel.Earth - lon.Earth) / smoothing
				latmag := latdel * latdel
				lonmag := londel * londel
				if latmag > lonmag && latmag > repose {
					// Latitudinal
					cel.Earth -= latdel
					lat.Earth += latdel
				} else if lonmag > repose {
					// Longitudinal
					cel.Earth -= londel
					lon.Earth += londel
				}
			}

			// Watershed
			{
				latdel := ((cel.Earth + cel.Water) - (lat.Earth + lat.Water)) / flowing
				londel := ((cel.Earth + cel.Water) - (lat.Earth + lat.Water)) / flowing
				// Clamp flow to avoid negative water.
				if latdel > cel.Water {
					latdel = cel.Water
				}
				if latdel < -lat.Water {
					latdel = -lat.Water
				}
				if londel > cel.Water {
					londel = cel.Water
				}
				if londel < -lon.Water {
					londel = -lon.Water
				}
				latmag := latdel * latdel
				lonmag := londel * londel
				if latmag > lonmag {
					cel.Water -= latdel
					lat.Water += latdel
				} else {
					cel.Water -= londel
					lon.Water += londel
				}
			}
		}
	}

	for pt.Y = 0; pt.Y < int(sim.scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.scale); pt.X++ {
			a := &next.Grid[sim.scale.Encode(pt)]
			next.EarthElevationStats.Add(a.Earth)
			next.WaterElevationStats.Add(a.Earth + a.Water)
			next.WaterStats.Add(a.Water)
		}
	}
}
