package bottletectonic

import (
	"borkshop/bottle"
	"borkshop/bottlepid"
	"borkshop/hilbert"
	"image"
	"math"
	"math/rand"
)

var vectors [bottle.NumPlates]image.Point

const unit = 0x10000

func init() {
	slice := math.Pi * 2 / float64(bottle.NumPlates)
	for i := 0; i < bottle.NumPlates; i++ {
		angle := slice * float64(i)
		vectors[i] = image.Point{
			X: int(math.Cos(angle) * unit),
			Y: int(math.Sin(angle) * unit),
		}
	}
}

type Quakes struct {
	Scale      hilbert.Scale
	Magnitude  int
	Controller bottlepid.Controller
	Disabled   bool
}

var _ bottle.Ticker = (*Quakes)(nil)

func (sim *Quakes) Tick(next, prev *bottle.Generation) {
	if sim.Disabled {
		return
	}

	sim.Controller.Tick(&next.ElevationSpreadController, &prev.ElevationSpreadController, prev.EarthElevationStats.Spread())
	control := next.ElevationSpreadController.Control

	var pt image.Point
	for pt.Y = 0; pt.Y < int(sim.Scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.Scale); pt.X++ {
			if rand.Intn(sim.Controller.Max) < control {
				cel := &next.Grid[sim.Scale.Encode(pt)]
				lat := &next.Grid[sim.Scale.Encode(pt.Add(image.Pt(1, 0)))]
				lon := &next.Grid[sim.Scale.Encode(pt.Add(image.Pt(0, 1)))]
				vector := vectors[cel.Plate]
				total := mag(vector.X) + mag(vector.Y)
				if rand.Intn(total) < mag(vector.X) {
					latdel := clamp(vector.X, -sim.Magnitude, sim.Magnitude)
					cel.Earth -= latdel
					lat.Earth += latdel
					next.QuakeFlow += mag(latdel)
				} else {
					londel := clamp(vector.Y, -sim.Magnitude, sim.Magnitude)
					cel.Earth -= londel
					lon.Earth += londel
					next.QuakeFlow += mag(londel)
				}
			}
		}
	}
}

func mag(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func clamp(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
