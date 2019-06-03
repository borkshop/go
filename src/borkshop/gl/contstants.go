// +build js

package gl

import (
	"fmt"
	"syscall/js"
)

// Constants provides access to appropriate named constants for a GL context.
type Constants struct {
	consts map[string]int
}

// Constant represents a webgl constant for some GL context.
type Constant struct {
	Name  string
	Value int
}

// Constant looks up and returns the named constant, panicing if none is
// defined for the given name.
func (con Constants) Constant(name string) Constant {
	if value, defined := con.consts[name]; defined {
		return Constant{name, value}
	}
	panic(fmt.Sprintf("undefined GL constant %s", name))
}

// ConstantByVal performs a reverse constant lookup by value, panicing if no
// constant is defined for the given value.
func (con Constants) ConstantByVal(value int) Constant {
	for name, val := range con.consts {
		if value == val {
			return Constant{name, value}
		}
	}
	panic(fmt.Sprintf("undefined GL constant for value %v", value))
}

var loadedGLConstants map[string]Constants

func constantsFor(gl js.Value) Constants {
	glProto := gl.Get("__proto__")
	name := glProto.Get("constructor").Get("name").String()
	con, defined := loadedGLConstants[name]
	if defined {
		return con
	}

	keys := js.Global().Get("Object").Call("keys", glProto)

	keyStrings := make([]string, 0, keys.Length())
	for i := 0; i < keys.Length(); i++ {
		if key := keys.Index(i).String(); isCaptialSnake(key) {
			if gl.Get(key).Type() == js.TypeNumber {
				keyStrings = append(keyStrings, key)
			}
		}
	}

	con.consts = make(map[string]int, 2*len(keyStrings))
	for _, key := range keyStrings {
		con.consts[key] = gl.Get(key).Int()
	}

	if loadedGLConstants == nil {
		loadedGLConstants = make(map[string]Constants)
	}
	loadedGLConstants[name] = con

	return con
}

func (c Constant) String() string {
	return fmt.Sprintf("gl.%s", c.Name)
}

func isCaptialSnake(s string) bool {
	for i := 0; i < len(s); i++ {
		switch {
		case 'A' <= s[i] && s[i] <= 'Z':
		case '0' <= s[i] && s[i] <= '9':
		case s[i] == '_':
		default:
			return false
		}
	}
	return true
}

type Type uint8

const (
	UnsignedByte Type = iota
	UnsignedShort
	UnsignedInt
)

func (typ Type) String() string {
	switch typ {
	case UnsignedByte:
		return "UNSIGNED_BYTE"
	case UnsignedShort:
		return "UNSIGNED_SHORT"
	case UnsignedInt:
		return "UNSIGNED_INT"
	default:
		return fmt.Sprintf("INVALID_TYPE(%02x)", uint8(typ))
	}
}
