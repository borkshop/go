// +build js

package gl

import "fmt"

// Attrib tracks a location to an attribute in a Program.
type Attrib struct {
	name   string
	loc    int
	size   int
	glType Constant
}

func (attr Attrib) String() string {
	return fmt.Sprintf("Attrib(%q, size:%v, type:%v)", attr.name, attr.size, attr.glType)
}

// GetAttrib returns an Attrib for the given named program attribute, or an
// error if no such attribute is found.
func (prog *Program) GetAttrib(name string) (attr Attrib, err error) {
	attr.name = name
	attr.loc = prog.gl.Call("getAttribLocation", prog.handle, attr.name).Int()
	if attr.loc < 0 {
		err = fmt.Errorf("no attrib location %q", attr.name)
	} else {
		info := prog.gl.Call("getActiveAttrib", prog.handle, attr.loc)
		attr.size = info.Get("size").Int()
		attr.glType = prog.ConstantByVal(info.Get("type").Int())
	}
	return attr, err
}

// EnableAttrib turns on the a generic vertex attribute array.
func (prog *Program) EnableAttrib(attr Attrib) {
	prog.gl.Call("enableVertexAttribArray", attr.loc)
}

// DisableAttrib turns on the a generic vertex attribute array.
func (prog *Program) DisableAttrib(attr Attrib) {
	prog.gl.Call("disableVertexAttribArray", attr.loc)
}

// AttribPointer binds the buffer currently bound to ArrayBuffer target to a
// generic vertex attribute of the current vertex buffer object and specifies
// its layout.
// func (prog *Program) AttribPointer(attr Attrib, normalized bool, stride, offset uint)

// (index, size, type, normalized, stride, offset)

// bindAttribLocation()
// getActiveAttrib()

// getContextAttributes()

// getVertexAttrib()
// getVertexAttribOffset()

// vertexAttrib[1234]f[v]()
