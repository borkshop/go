// +build js

package gl

import (
	"fmt"
	"syscall/js"
)

type Uniform struct {
	name   string
	loc    js.Value
	size   int
	glType Constant
}

func (uni Uniform) String() string {
	return fmt.Sprintf("Uniform(%q, size:%v, type:%v)", uni.name, uni.size, uni.glType)
}

// GetUniform returns an Uniform for the given named program uniform, or an
// error if no such uniform is found.
func (prog *Program) GetUniform(name string) (uni Uniform, err error) {
	uni.name = name
	uni.loc = prog.gl.Call("getUniformLocation", prog.handle, uni.name)
	if !uni.loc.Truthy() {
		err = fmt.Errorf("no uniform location %q", uni.name)
	} else {
		info := prog.gl.Call("getActiveUniform", prog.handle, uni.loc)
		uni.size = info.Get("size").Int()
		uni.glType = prog.ConstantByVal(info.Get("type").Int())
	}
	return uni, err
}

// getUniform()
// uniformMatrix[234]fv()
// uniform[1234][fi][v]()
