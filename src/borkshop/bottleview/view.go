package bottleview

import (
	"borkshop/bottle"
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

// View draws generations of the bottle world onto a screen.
type View interface {
	Draw(screen *anansi.Screen, rect ansi.Rectangle, gen *bottle.Generation, gp image.Point)
}
