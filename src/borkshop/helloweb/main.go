// +build js

package main

import "syscall/js"

func main() {
	document := js.Global().Get("document")
	body := document.Get("body")
	textNode := document.Call("createTextNode", "Hello, Web!")
	body.Call("appendChild", textNode)
}
