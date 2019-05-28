package bottletower

import (
	"borkshop/bottle"
	"borkshop/hilbert"
)

type Resetter struct {
	Scale hilbert.Scale
}

func (r Resetter) Reset(gen *bottle.Generation) {
	gen.Grid[int(r.Scale*r.Scale)/2].Earth = 100
}
