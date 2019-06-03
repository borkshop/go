// +build js

package gl

import (
	"errors"
	"image"
	"syscall/js"
)

type Canvas struct {
	el js.Value
	GL
}

func (can *Canvas) Init(el js.Value) error {
	if !el.Truthy() {
		return errors.New("no <canvas> element given")
	}
	can.el = el
	return can.GL.Init(el)
}

// Size returns the size of the <canvas> element.
func (can *Canvas) Size() image.Point {
	width := can.el.Get("width").Int()
	height := can.el.Get("height").Int()
	return image.Pt(width, height)
}

// Resize the <canvas> element.
func (can *Canvas) Resize(size image.Point) {
	can.el.Set("width", size.X)
	can.el.Set("height", size.Y)
	can.gl.Call("viewport", 0, 0, size.X, size.Y)
}
