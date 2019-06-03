// +build js

package gl

import (
	"errors"
	"image"
	"image/color"
	"syscall/js"
)

// GL is a webgl rendering context attached to some <canvas> element in the
// dom.
type GL struct {
	gl js.Value // TODO rename handle or context
	Constants

	shaders map[ShaderSource]js.Value
	progs   []Program
}

// Init initalizes a webgl context under the given <canvas> element,
// populating Constants, and preparing to build shader programs.
func (gl *GL) Init(el js.Value) error {
	if !el.Truthy() {
		return errors.New("no <canvas> element given")
	}

	for _, contextType := range []string{
		// TODO webgl2
		"webgl",
		"experimental-webgl",
	} {
		ctx := el.Call("getContext", contextType)
		if ctx.Truthy() {
			gl.gl = ctx
			break
		}
	}
	if !gl.gl.Truthy() {
		return errors.New("unable to get webgl context")
	}

	gl.glConstants = constantsFor(gl.gl)

	gl.shaders = make(map[ShaderSource]js.Value)
	return nil
}

// Release all built programs, and delete their compiled shaders.
func (gl *GL) Release() {
	for i := range gl.progs {
		gl.progs[i].Release()
	}
	gl.progs = gl.progs[:0]
	for key, shader := range gl.shaders {
		gl.gl.Call("deleteShader", shader)
		delete(gl.shaders, key)
	}
}

// Finish blocks execution until all previously called commands are finished.
func (gl *GL) Finish() {
	gl.gl.Call("finish")
}

// Flush empties different buffer commands, causing all commands to be executed
// as quickly as possible.
func (gl *GL) Flush() {
	gl.gl.Call("flush")
}

// DrawingBufferSize represents the actual size of the current drawing buffer.
// It should match the height attribute of the <canvas> element, but might
// differ if the implementation is not able to provide the requested height.
func (gl *GL) DrawingBufferSize() image.Point {
	return image.Pt(
		gl.gl.Get("drawingBufferHeight").Int(),
		gl.gl.Get("drawingBufferWidth").Int(),
	)
}

// ClearColor specifies the color values used when clearing color buffers.
//
// This specifies what color values to use when calling the clear() method. The
// values are clamped between 0 and 1.
func (gl *GL) ClearColor(c color.Color) {
	const max = 0xffff
	r, g, b, a := c.RGBA()
	gl.gl.Call("clearColor", float32(r)/max, float32(g)/max, float32(b)/max, float32(a)/max)
}

// ClearDepth specifies the clear value for the depth buffer.
//
// This specifies what depth value to use when calling the clear() method. The
// value is clamped between 0 and 1.
func (gl *GL) ClearDepth(d float32) {
	gl.gl.Call("clearDepth", d)
}

// ClearStencil specifies the clear value for the stencil buffer.
//
// This specifies what stencil value to use when calling the clear() method.
func (gl *GL) ClearStencil(s float32) {
	gl.gl.Call("clearStencil", s)
}

// Clear clears buffers to preset values.
//
// The preset values can be set by ClearColor(), ClearDepth() or
// ClearStencil().
//
// The scissor box, dithering, and buffer writemasks can affect the clear()
// method.
func (gl *GL) Clear(clearBits ClearBit) {
	gl.gl.Call("clear", clearBits.get(gl.Constants))
}

type ClearBit uint8

const (
	ColorBufferBit ClearBit = 1 << iota
	DepthBufferBit
	StencilBufferBit
)

func (clr ClearBit) get(con Constants) int {
	var mask int
	if clr&ColorBufferBit != 0 {
		mask |= con.Constant("COLOR_BUFFER_BIT").Value
	}
	if clr&DepthBufferBit != 0 {
		mask |= con.Constant("DEPTH_BUFFER_BIT").Value
	}
	if clr&StencilBufferBit != 0 {
		mask |= con.Constant("STENCIL_BUFFER_BIT").Value
	}
	return mask
}
