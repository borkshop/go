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

type ShaderSource struct {
	Type   string
	Source string
}

func VertexShader(source string) ShaderSource   { return ShaderSource{"VERTEX_SHADER", source} }
func FragmentShader(source string) ShaderSource { return ShaderSource{"FRAGMENT_SHADER", source} }

type Canvas struct {
	el js.Value
	gl js.Value

	shaders map[ShaderSource]js.Value
	progs   []Program
}

func (can *Canvas) Init(el js.Value) error {
	if !el.Truthy() {
		return errors.New("no canvas element given")
	}
	can.el = el

	for _, contextType := range []string{
		// TODO webgl2
		"webgl",
		"experimental-webgl",
	} {
		gl := el.Call("getContext", contextType)
		if gl.Truthy() {
			can.gl = gl
			break
		}
	}
	if can.gl.Truthy() {
		return errors.New("unable to get webgl context")
	}

	return nil
}

func (can *Canvas) Release() {
	for i := range can.progs {
		can.progs[i].Release()
	}
	can.progs = can.progs[:0]

	for key, shader := range can.shaders {
		can.gl.Call("deleteShader", shader)
		delete(can.shaders, key)
	}
}

func (can *Canvas) Size() image.Point {
	width := can.el.Get("width").Int()
	height := can.el.Get("height").Int()
	return image.Pt(width, height)
}

func (can *Canvas) Resize(size image.Point) {
	can.el.Set("width", size.X)
	can.el.Set("height", size.Y)
}

func (can *Canvas) Build(srcs ...ShaderSource) (prog Program, err error) {
	// TODO dedupe?
	prog.gl = can.gl
	prog.srcs = srcs
	prog.handle = can.gl.Call("createProgram")

	for _, src := range prog.srcs {
		shader, err := can.compile(src)
		if err != nil {
			return Program{}, err
		}
		prog.attach(shader)
	}

	err = prog.link()
	return prog, err
}

func (can *Canvas) compile(src ShaderSource) (js.Value, error) {
	shader, defined := can.shaders[src]
	if !defined {
		shader = can.gl.Call("createShader", can.gl.Get(src.Type))
		can.gl.Call("shaderSource", shader, src)
		can.gl.Call("compileShader", shader)
		if !can.gl.Call("getShaderParameter", shader, can.gl.Get("COMPILE_STATUS")).Bool() {
			return shader, fmt.Errorf("could not compile %s: %v", src.Type,
				can.gl.Call("getShaderInfoLog", shader).String())
		}
		can.shaders[src] = shader
	}
	return shader, nil
}

func (prog Program) attach(shader js.Value) {
	prog.gl.Call("attachShader", prog.handle, shader)
}

func (prog Program) link() error {
	prog.gl.Call("linkProgram", prog.handle)
	prog.gl.Call("validateProgram", prog.handle)
	if !prog.gl.Call("getProgramParameter", prog.handle, prog.gl.Get("LINK_STATUS")).Bool() {
		infoLog := prog.gl.Call("getProgramInfoLog", prog.handle)
		return fmt.Errorf("could not link program: %v", infoLog.String())
	}
	return nil
}

type Program struct {
	gl     js.Value
	handle js.Value
	srcs   []ShaderSource
}

func (prog *Program) Release() {
	if prog.handle != js.Undefined() {
		prog.gl.Call("deleteProgram", prog.handle)
		prog.handle = js.Undefined()
	}
}

func (prog *Program) Bind(inst interface{}) error {
	val := reflect.ValueOf(inst)
	if val.Kind() != reflect.Struct {
		return errors.New("gl.Program binding instance must be a struct")
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

func (prog *Program) DrawArrays(mode Mode, first, count int) error {
	modeVal, err := mode.get(prog.gl)
	if err == nil {
		prog.gl.Call("drawArrays", modeVal, first, count)
	}
	return err
}

func (prog *Program) DrawElements(mode Mode, count int, typ Type, offset uint) error {
	modeVal, err := mode.get(prog.gl)
	if err == nil {
		typVal, err := typ.get(prog.gl)
		if err == nil {
			prog.gl.Call("drawElements", modeVal, count, typVal, offset)
		}
	}
	return err
}

// TODO webgl2 drawing routines

func (mode Mode) String() string {
	switch mode {
	case Points:
		return "Points"
	case LineStrip:
		return "LineStrip"
	case LineLoop:
		return "LineLoop"
	case Lines:
		return "Lines"
	case TriangleStrip:
		return "TriangleStrip"
	case TriangleFan:
		return "TriangleFan"
	case Triangles:
		return "Triangles"
	default:
		return fmt.Sprintf("InvalidMode(%02x)", uint8(mode))
	}
}

func (mode Mode) get(gl js.Value) (val js.Value, err error) {
	switch mode {
	case Points:
		val = gl.Get("POINTS")
	case LineStrip:
		val = gl.Get("LINE_STRIP")
	case LineLoop:
		val = gl.Get("LINE_LOOP")
	case Lines:
		val = gl.Get("LINES")
	case TriangleStrip:
		val = gl.Get("TRIANGLE_STRIP")
	case TriangleFan:
		val = gl.Get("TRIANGLE_FAN")
	case Triangles:
		val = gl.Get("TRIANGLES")
	}
	if val == js.Undefined() {
		err = fmt.Errorf("unsupported gl drawing mode %v", mode)
	}
	return val, err
}

func (typ Type) get(gl js.Value) (val js.Value, err error) {
	switch typ {
	case UnsignedByte:
		val = gl.Get("UNSIGNED_BYTE")
	case UnsignedShort:
		val = gl.Get("UNSIGNED_SHORT")
	case UnsignedInt:
		val = gl.Get("UNSIGNED_INT")
	}
	if val == js.Undefined() {
		err = fmt.Errorf("unsupported gl type %v", typ)
	}
	return val, err
}

func (typ Type) String() string {
	switch typ {
	case UnsignedByte:
		return "UnsignedByte"
	case UnsignedShort:
		return "UnsignedShort"
	case UnsignedInt:
		return "UnsignedInt"
	default:
		return fmt.Sprintf("InvalidType(%02x)", uint8(typ))
	}
}
