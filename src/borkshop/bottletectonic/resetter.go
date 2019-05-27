package bottletectonic

import (
	"borkshop/bottle"
	"math/rand"
)

// Resetter sets the levels of terrain to the values from a floating point map.
type Resetter struct{}

func (r Resetter) Reset(gen *bottle.Generation) {
	for i := 0; i < len(gen.Grid); i++ {
		gen.Grid[i].Plate = uint8(rand.Intn(bottle.NumPlates))
	}
}