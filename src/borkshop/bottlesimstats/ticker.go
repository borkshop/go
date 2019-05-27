package bottlesimstats

import "borkshop/bottle"

type Pre struct{}

func (Pre) Tick(next, prev *bottle.Generation) {
	next.Num = prev.Num + 1
	next.EarthElevationStats.Reset()
	next.WaterElevationStats.Reset()
	next.WaterStats.Reset()
	next.WaterCoverage = 0
	next.EarthFlow = 0
	next.WaterFlow = 0
	next.PlateSizes = [bottle.NumPlates]int{}
	for i := 0; i < len(next.Grid); i++ {
		next.Grid[i].Random = prev.Grid[i].Random
	}
}

type Post struct{}

func (Post) Tick(next, prev *bottle.Generation) {
	for i := 0; i < len(next.Grid); i++ {
		cell := &next.Grid[i]
		next.EarthElevationStats.Add(cell.Earth)
		next.WaterElevationStats.Add(cell.Earth + cell.Water)
		next.WaterStats.Add(cell.Water)
		if cell.Water > 2 {
			next.WaterCoverage++
		}
		next.PlateSizes[cell.Plate]++
	}
}
