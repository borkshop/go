package bottlemudslide

import (
	"borkshop/bottle"
	"borkshop/hilbert"
	"borkshop/repose"
	"image"
)

type Simulation struct {
	Scale  hilbert.Scale
	Repose int
	area   int
}

var _ bottle.Ticker = (*Simulation)(nil)

func (sim *Simulation) Tick(next, prev *bottle.Generation) {
	for i := 0; i < len(next.Grid); i++ {
		next.Grid[i].Earth = prev.Grid[i].Earth
	}

	var pt image.Point
	for pt.Y = 0; pt.Y < int(sim.Scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.Scale); pt.X++ {
			cel := &next.Grid[sim.Scale.Encode(pt)]
			lat := &next.Grid[sim.Scale.Encode(pt.Add(image.Pt(1, 0)))]
			lon := &next.Grid[sim.Scale.Encode(pt.Add(image.Pt(0, 1)))]

			// Settle slopes to angle of repose
			latdel := (cel.Earth - lat.Earth)
			londel := (cel.Earth - lon.Earth)
			latmag := mag(latdel)
			lonmag := mag(londel)

			var latlon LatOrLon
			switch {
			case latmag <= sim.Repose && lonmag <= sim.Repose:
				latlon = LatNorLon
			case latmag > sim.Repose && lonmag <= sim.Repose:
				latlon = Lat
			case latmag <= sim.Repose && lonmag > sim.Repose:
				latlon = Lon
			case latmag > lonmag:
				latlon = Lat
			case latmag < lonmag:
				latlon = Lon
			case cel.Random.Uint64()&1 == 0:
				latlon = Lat
			default:
				latlon = Lon
			}

			var flow int
			switch latlon {
			case Lat:
				cel.Earth, lat.Earth, flow = repose.Slide(cel.Earth, lat.Earth, sim.Repose)
				next.EarthFlow += flow
			case Lon:
				cel.Earth, lon.Earth, flow = repose.Slide(cel.Earth, lon.Earth, sim.Repose)
				next.EarthFlow += flow
			}
		}
	}
}

type LatOrLon int

const (
	LatNorLon LatOrLon = iota + 1
	Lat
	Lon
)

func mag(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
