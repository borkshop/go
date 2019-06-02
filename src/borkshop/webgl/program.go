package webgl

// +build js

import (
	"errors"
	"fmt"
	"image"
	"reflect"
	"strings"
	"syscall/js"
)

type Mode uint8

const (
	Points Mode = iota
	LineStrip
	LineLoop
	Lines
	TriangleStrip
	TriangleFan
	Triangles
)

type Type uint8

const (
	UnsignedByte Type = iota
	UnsignedShort
	UnsignedInt
)

var (
	errNoCanvas  = errors.New("no canvas element given")
	errNoContext = errors.New("unable to get webgl context")
)

func getContext(el js.Value) js.Value {
	for _, contextType := range []string{"webgl", "experimental-webgl"} {
		gl := el.Call("getContext", contextType)
		if gl.Truthy() {
			return gl
		}
	}
	return js.Value{}
}

func NewCanvas(el js.Value) (*Canvas, error) {
	if !el.Truthy() {
		return nil, errNoCanvas
	}

	gl := getContext(el)
	if gl.Truthy() {
		return nil, errNoContext
	}

	can := Canvas{
		can: el,
		par: el.Get("parentNode"),
		gl:  gl,
	}

	// TODO hookup resize event handler
	can.updateSize()

	return &can, nil
}

type Canvas struct {
	can js.Value
	par js.Value
	gl  js.Value

	shaders map[ShaderSource]js.Value
}

func (can *Canvas) updateSize() {
	width := can.par.Get("clientWidth").Int()
	height := can.par.Get("clientHeight").Int()
	can.can.Set("width", width)
	can.can.Set("height", height)
}

func (can *Canvas) Size() image.Point {
	width := can.can.Get("width").Int()
	height := can.can.Get("height").Int()
	return image.Pt(width, height)
}

type ShaderSource struct {
	Type   string
	Source string
}

func VertexShader(source string) ShaderSource   { return ShaderSource{"VERTEX_SHADER", source} }
func FragmentShader(source string) ShaderSource { return ShaderSource{"FRAGMENT_SHADER", source} }

func (can *Canvas) compile(src ShaderSource) (js.Value, error) {
	handle, defined := can.shaders[src]
	if !defined {
		handle = can.gl.Call("createShader", can.gl.Get(src.Type))
		can.gl.Call("shaderSource", handle, src)
		can.gl.Call("compileShader", handle)
		if !can.gl.Call("getShaderParameter", handle, can.gl.Get("COMPILE_STATUS")).Bool() {
			return handle, fmt.Errorf("could not compile %s: %v", src.Type,
				can.gl.Call("getShaderInfoLog", handle).String())
		}
		can.shaders[src] = handle
	}
	return handle, nil
}

func (can *Canvas) BuildProgram(srcs ...ShaderSource) (Program, error) {
	prog := can.gl.Call("createProgram")
	for _, src := range srcs {
		shader, err := can.compile(src)
		if err != nil {
			return Program{}, err
		}
		can.gl.Call("attachShader", prog, shader)
	}

	var err error
	p := Program{can.gl, prog, srcs}
	can.gl.Call("linkProgram", p.handle)
	can.gl.Call("validateProgram", p.handle)

	if !can.gl.Call("getProgramParameter", p.handle, can.gl.Get("LINK_STATUS")).Bool() {
		infoLog := can.gl.Call("getProgramInfoLog", p.handle)
		err = fmt.Errorf("could not link program: %v", infoLog.String())
	}
	return p, err
}

type Program struct {
	gl     js.Value
	handle js.Value
	srcs   []ShaderSource
}

func (prog *Program) Init(inst interface{}) error {
	val := reflect.ValueOf(inst)
	if val.Kind() != reflect.Struct {
		return errors.New("prog must be a struct")
	}

	// TODO integrate into building, reflect for source fields

	prog.gl.Call("useProgram", prog.handle)

	typ := reflect.TypeOf(inst)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		tag := field.Tag.Get("gl")

		if strings.HasPrefix(tag, "attrib:") {
			name := tag[7:]
			handle := prog.gl.Call("getAttribLocation", prog.handle, name)
			if !handle.Truthy() {
				return fmt.Errorf("no attrib location %q for field %q", name, field.Name)
			}
			val.FieldByIndex(field.Index).Set(handle)
			continue
		}

		if strings.HasPrefix(tag, "uniform:") {
			name := tag[8:]
			handle := prog.gl.Call("getUniformLocation", prog.handle, name)
			if !handle.Truthy() {
				return fmt.Errorf("no uniform location %q for field %q", name, field.Name)
			}
			val.FieldByIndex(field.Index).Set(handle)
			continue
		}

		if tag != "" {
			return fmt.Errorf("invalid gl tag %q", tag)
		}
	}
	return nil
}

func (prog *Program) Use() {
	prog.gl.Call("useProgram", prog.handle)
}

func (prog *Program) DrawArrays(mode Mode, first, count int) {
	prog.gl.Call("drawArrays", mode, first, count)
}

func (prog *Program) DrawElements(mode Mode, count int, typ Type, offset uint) {
	prog.gl.Call("drawElements", mode, count, typ, offset)
}

/*

	ParametersSection
	mode

	first A GLint specifying the starting index in the array of vector points.
	count A GLsizei specifying the number of indices to be rendered.

	ParametersSection
	mode

	count A GLsizei specifying the number of elements to be rendered.

	type
	A GLenum specifying the type of the values in the element array buffer. Possible values are:
	gl.UNSIGNED_BYTE
	gl.UNSIGNED_SHORT
	When using the OES_element_index_uint extension:
	gl.UNSIGNED_INT

	offset A GLintptr specifying a byte offset in the element array buffer. Must be a valid multiple of the size of the given type.

*/
