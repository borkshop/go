package bottlewatershed

import (
	"borkshop/bottle"
	"borkshop/hilbert"
	"image"
)

type Simulation struct {
	Scale hilbert.Scale
}

var _ bottle.Ticker = (*Simulation)(nil)

func (sim *Simulation) Tick(next, prev *bottle.Generation) {
	for i := 0; i < len(next.Grid); i++ {
		next.Grid[i].Water = prev.Grid[i].Water
	}

	var pt image.Point
	for pt.Y = 0; pt.Y < int(sim.Scale); pt.Y++ {
		for pt.X = 0; pt.X < int(sim.Scale); pt.X++ {
			cel := &next.Grid[sim.Scale.Encode(pt)]
			lat := &next.Grid[sim.Scale.Encode(pt.Add(image.Pt(1, 0)))]
			lon := &next.Grid[sim.Scale.Encode(pt.Add(image.Pt(0, 1)))]

			// Settle slopes to angle of repose
			{
				latdel := ((cel.Earth + cel.Water) - (lat.Earth + lat.Water))
				londel := ((cel.Earth + cel.Water) - (lon.Earth + lon.Water))
				latmag := mag(latdel)
				lonmag := mag(londel)

				var latlon LatOrLon
				switch {
				case latmag == lonmag:
					latlon = LatNorLon
				case latmag > lonmag:
					latlon = Lat
				case latmag < lonmag:
					latlon = Lon
				case cel.Random.Uint64()&1 == 0:
					latlon = Lat
				default:
					latlon = Lon
				}

				switch latlon {
				case Lat:
					latdel = clamp(latdel/2, -lat.Water, cel.Water)
					cel.Water -= latdel
					lat.Water += latdel
					next.WaterFlow += mag(latdel)
				case Lon:
					londel = clamp(londel/2, -lon.Water, cel.Water)
					cel.Water -= londel
					lon.Water += londel
					next.WaterFlow += mag(londel)
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

type LatOrLon int

const (
	LatNorLon LatOrLon = iota + 1
	Lat
	Lon
)
