package bottlewatercoverage

import (
	"borkshop/bottle"
	"borkshop/bottlepid"
)

type Simulation struct {
	Controller bottlepid.Controller
}

var _ bottle.Ticker = (*Simulation)(nil)

func (sim *Simulation) Tick(next, prev *bottle.Generation) {
	sim.Controller.Tick(&next.WaterCoverageController, &prev.WaterCoverageController, prev.WaterCoverage)
	control := next.WaterCoverageController.Control
	switch {
	case control > 0:
		for i := 0; i < len(next.Grid); i++ {
			if int(next.Grid[i].Random.Uint64()&0xffffff) < control {
				next.Grid[i].Water += 1
			}
		}
	case control < 0:
		for i := 0; i < len(next.Grid); i++ {
			if int(next.Grid[i].Random.Uint64()&0xfffff) < -control {
				if next.Grid[i].Water > 0 {
					next.Grid[i].Water -= 1
				}
			}
		}
	}
}
