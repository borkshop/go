// +build js

package gl

import (
	"fmt"
	"syscall/js"
)

// BufferTarget specifies a Buffer binding point (target).
type BufferTarget uint8

const (
	// Buffer containing vertex attributes, such as vertex coordinates, texture
	// coordinate data, or vertex color data.
	ArrayBuffer BufferTarget = iota

	// Buffer used for element indices.
	ElementArrayBuffer

	// TODO additional webgl2 values
)

// BufferUsage specifies the usage pattern of a Buffer.
type BufferUsage uint8

const (
	// Contents of the buffer are likely to be used often and not change often.
	// Contents are written to the buffer, but not read.
	StaticDraw BufferUsage = iota

	// Contents of the buffer are likely to be used often and change often.
	// Contents are written to the buffer, but not read.
	DynamicDraw

	// Contents of the buffer are likely to not be used often. Contents are
	// written to the buffer, but not read.
	StreamDraw

	// TODO additional webgl2 values
)

// Buffer tracks a handle to a WebGLBuffer.
type Buffer struct {
	handle   js.Value
	glTarget Constant
	glUsage  Constant
}

func (buf Buffer) String() string {
	return fmt.Sprintf("Buffer(target:%v usage:%v)", buf.glTarget, buf.glUsage)
}

// CreateBuffer creates and initializes a WebGLBuffer storing data such as
// vertices or colors.
func (prog *Program) CreateBuffer(targ BufferTarget, usage BufferUsage) Buffer {
	glTarget := prog.Constant(targ.String())
	glUsage := prog.Constant(usage.String())
	return Buffer{prog.gl.Call("createBuffer"), glTarget, glUsage}
}

// DeleteBuffer deletes a given WebGLBuffer. This method has no effect if the
// buffer has already been deleted.
func (prog *Program) DeleteBuffer(buf Buffer) {
	prog.gl.Call("deleteBuffer", buf.handle)
}

// BufferData initializes and creates the buffer object's data store.
//
// srcData must be an ArrayBuffer, SharedArrayBuffer or one of the
// ArrayBufferView typed array types that will be copied into the data store.
// If null, a data store is still created, but the content is uninitialized and
// undefined.
//
// TODO webgl2 variant
func (prog *Program) BufferData(buf Buffer, srcData js.Value) {
	prog.gl.Call("bindBuffer", buf.glTarget.Value, buf.handle)
	prog.gl.Call("bufferData", buf.glTarget.Value, srcData, buf.glUsage.Value)
}

// BufferSubData updates a subset of a buffer object's data store.
//
// srcData must be an ArrayBuffer, SharedArrayBuffer or one of the
// ArrayBufferView typed array types that will be copied into the data store.
//
// TODO webgl2 variant
func (prog *Program) BufferSubData(buf Buffer, offset uint, srcData js.Value) {
	prog.gl.Call("bindBuffer", buf.glTarget.Value, buf.handle)
	prog.gl.Call("bufferSubData", buf.glTarget.Value, offset, srcData)
}

func (targ *BufferTarget) Set(s string) error {
	switch s {
	case "":
	case "ARRAY_BUFFER":
		*targ = ArrayBuffer
	case "ELEMENT_ARRAY_BUFFER":
		*targ = ElementArrayBuffer
	default:
		return fmt.Errorf("invalid gl.BufferTarget=%q", s)
	}
	return nil
}

func (usage *BufferUsage) Set(s string) error {
	switch s {
	case "":
	case "STATIC_DRAW":
		*usage = StaticDraw
	case "DYNAMIC_DRAW":
		*usage = DynamicDraw
	case "STREAM_DRAW":
		*usage = StreamDraw
	default:
		return fmt.Errorf("invalid gl.BufferUsage=%q", s)
	}
	return nil
}

func (targ BufferTarget) String() string {
	switch targ {
	case ArrayBuffer:
		return "ARRAY_BUFFER"
	case ElementArrayBuffer:
		return "ELEMENT_ARRAY_BUFFER"
	}
	return fmt.Sprintf("INVALID_BUFFER_TARGET(%02x)", uint8(targ))
}

func (usage BufferUsage) String() string {
	switch usage {
	case StaticDraw:
		return "STATIC_DRAW"
	case DynamicDraw:
		return "DYNAMIC_DRAW"
	case StreamDraw:
		return "STREAM_DRAW"
	}
	return fmt.Sprintf("INVALID_BUFFER_USAGE(%02x)", uint8(usage))
}
