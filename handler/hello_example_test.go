package handler_test

// +build js

import (
	"syscall/js"
)

var (
	window = js.Global().Get("window")
	document = js.Global().Get("document")
)

func ExampleHello() {
	window.Call("addEventListener", "DOMContentLoaded", js.FuncOf(contentLoaded))
}

func contentLoaded(this js.Value, args []js.Value) interface{} {
	document.Get("body").Set("innerText", "hello world")
}
