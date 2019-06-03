// +build js

package gl

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"syscall/js"
)

type ShaderType uint8

const (
	VertexShaderType ShaderType = iota + 1
	FragmentShaderType
)

type ShaderSource struct {
	Type   ShaderType
	Source string
}

func VertexShader(source string) ShaderSource   { return ShaderSource{VertexShaderType, source} }
func FragmentShader(source string) ShaderSource { return ShaderSource{FragmentShaderType, source} }

// Build compiles and links the given shaders into a new Program handle.
func (gl *GL) Build(sources ...ShaderSource) (Program, error) {
	prog := Program{
		GL:      gl,
		Sources: sources,
		handle:  gl.gl.Call("createProgram"),
	}

	for _, src := range prog.Sources {
		shader, err := gl.compile(src)
		if err != nil {
			return Program{}, err
		}
		prog.attach(shader)
	}

	if err := prog.link(); err != nil {
		prog.gl.Call("deleteProgram", prog.handle)
		return Program{}, err
	}

	gl.progs = append(gl.progs, prog)
	return prog, nil
}

func (gl *GL) compile(src ShaderSource) (js.Value, error) {
	shader, defined := gl.shaders[src]
	if !defined {
		shader = gl.gl.Call("createShader", gl.Constant(src.Type.String()).Value)
		gl.gl.Call("shaderSource", shader, src.Source)
		gl.gl.Call("compileShader", shader)
		if !gl.gl.Call("getShaderParameter", shader, gl.gl.Get("COMPILE_STATUS")).Bool() {
			return shader, fmt.Errorf("could not compile %v: %v", src.Type,
				gl.gl.Call("getShaderInfoLog", shader).String())
		}
		gl.shaders[src] = shader
	}
	return shader, nil
}

type Program struct {
	*GL
	handle  js.Value
	Sources []ShaderSource
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

func (prog *Program) Release() {
	if prog.handle != js.Undefined() {
		prog.gl.Call("deleteProgram", prog.handle)
		prog.handle = js.Undefined()
	}
}

func (prog *Program) Bind(inst interface{}) error {
	ps := reflect.ValueOf(inst)
	if ps.Kind() != reflect.Ptr {
		return errors.New("gl.Program binding instance must be a struct pointer")
	}

	s := ps.Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("gl.Program binding instance must be a struct pointer")
	}

	prog.gl.Call("useProgram", prog.handle)

	uniformType := reflect.TypeOf(Uniform{})
	attrType := reflect.TypeOf(Attrib{})
	bufType := reflect.TypeOf(Buffer{})

	typ := s.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if err := func() error {
			fieldVal := s.FieldByIndex(field.Index)
			var val reflect.Value

			switch field.Type {

			case uniformType:
				name := field.Name
				if tag := field.Tag.Get("glName"); tag != "" {
					name = tag
				}
				uni, err := prog.GetUniform(name)
				if err != nil {
					return err
				}
				val = reflect.ValueOf(uni)

			case attrType:
				name := field.Name
				if tag := field.Tag.Get("glName"); tag != "" {
					name = tag
				}
				attr, err := prog.GetAttrib(name)
				if err != nil {
					return err
				}
				val = reflect.ValueOf(attr)

			case bufType:
				var targ BufferTarget
				if err := targ.Set(field.Tag.Get("glTarget")); err != nil {
					return err
				}
				var usage BufferUsage
				if err := usage.Set(field.Tag.Get("glUsage")); err != nil {
					return err
				}
				buf := prog.CreateBuffer(targ, usage)
				val = reflect.ValueOf(buf)

			default:
				return nil

			}

			if !fieldVal.CanSet() {
				return errors.New("cannot set field")
			}
			fieldVal.Set(val)
			log.Printf("built %v => %v", field.Name, val)
			return nil
		}(); err != nil {
			return fmt.Errorf("unable to bind gl.Program field %v: %v", field.Name, err)
		}

	}
	return nil
}

func (prog *Program) Use() {
	prog.gl.Call("useProgram", prog.handle)
}

type DrawMode uint8

const (
	DrawPoints DrawMode = iota
	DrawLineStrip
	DrawLineLoop
	DrawLines
	DrawTriangleStrip
	DrawTriangleFan
	DrawTriangles
)

func (prog *Program) DrawArrays(mode DrawMode, first, count int) {
	glDraw := prog.Constant(mode.String()).Value
	prog.gl.Call("drawArrays", glDraw, first, count)
}

func (prog *Program) DrawElements(mode DrawMode, count int, typ Type, offset uint) {
	glDraw := prog.Constant(mode.String()).Value
	glType := prog.Constant(typ.String()).Value
	prog.gl.Call("drawElements", glDraw, count, glType, offset)
}

// TODO webgl2 drawing routines

func (mode DrawMode) String() string {
	switch mode {
	case DrawPoints:
		return "DRAW_POINTS"
	case DrawLineStrip:
		return "DRAW_LINE_STRIP"
	case DrawLineLoop:
		return "DRAW_LINE_LOOP"
	case DrawLines:
		return "DRAW_LINES"
	case DrawTriangleStrip:
		return "DRAW_TRIANGLE_STRIP"
	case DrawTriangleFan:
		return "DRAW_TRIANGLE_FAN"
	case DrawTriangles:
		return "DRAW_TRIANGLES"
	default:
		return fmt.Sprintf("INVALID_MODE(%02x)", uint8(mode))
	}
}

func (st ShaderType) String() string {
	switch st {
	case VertexShaderType:
		return "VERTEX_SHADER"
	case FragmentShaderType:
		return "fragment_shader"
	default:
		return fmt.Sprintf("INVALID_SHADER_TYPE(%02x)", uint8(st))
	}
}
