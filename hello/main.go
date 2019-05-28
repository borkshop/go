// +build js

// package main implements a simple hello prompter.
package main

import (
	"fmt"
	"os"
	"syscall/js"
)

var document = js.Global().Get("document")

func main() {
	input := document.Call("querySelector", os.Getenv("input"))
	if !input.Truthy() {
		panic("missing $input element")
	}

	output := document.Call("querySelector", os.Getenv("output"))
	if !output.Truthy() {
		panic("missing $output element")
	}

	input.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		val := input.Get("value")
		output.Set("innerText", fmt.Sprintf("Hello %q", val))
		return nil
	}))

	select {} // hang around to process events
}
