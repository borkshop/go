package bottletectonic

import (
	"borkshop/bottle"
	"borkshop/hilbert"
	"image"
	"math/rand"
)

type Simulation struct {
	Scale hilbert.Scale
}

var _ bottle.Ticker = (*Simulation)(nil)

func (sim *Simulation) Tick(next, prev *bottle.Generation) {
	for i := 0; i < len(next.Grid); i++ {
		next.Grid[i].Earth = prev.Grid[i].Earth
	}

	var pt image.Point
	for pt.Y = 0; pt.Y < int(sim.Scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.Scale); pt.X++ {
			i := sim.Scale.Encode(pt)

			// Read the neighborhood.
			neighborhood := []uint8{
				prev.Grid[i].Plate,
				prev.Grid[sim.Scale.Encode(pt.Add(image.Pt(-1, 0)))].Plate,
				prev.Grid[sim.Scale.Encode(pt.Add(image.Pt(1, 0)))].Plate,
				prev.Grid[sim.Scale.Encode(pt.Add(image.Pt(0, -1)))].Plate,
				prev.Grid[sim.Scale.Encode(pt.Add(image.Pt(0, 1)))].Plate,
			}

			// Construct a histogram: the number of tickets each tectonic plate
			// enters in the lottery for the next generation of this cell.
			var weights [bottle.NumPlates]int
			total := 0
			for i := 0; i < len(neighborhood); i++ {
				weights[neighborhood[i]]++
			}
			for i := 0; i < bottle.NumPlates; i++ {
				weight := weights[i]
				// The weight of each plate type is the square of the number of
				// neighbors of that color, times how rare the color is
				// globally in the previous generation.
				weights[i] = weight * weight * (int(sim.Scale*sim.Scale) - prev.PlateSizes[i])
				total += weights[i]
			}

			// Chose a random plate, weighted by local and global statistics.
			choice := int(rand.Int63() % int64(total))
			var plate uint8
			thresh := 0
			for ; plate < bottle.NumPlates; plate++ {
				thresh += weights[plate]
				if choice < thresh {
					next.Grid[i].Plate = plate
					break
				}
			}

		}
	}
}
