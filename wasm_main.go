// +build wasm js

package main

import (
	"fmt"
	"log"
	"os"
	"syscall/js"
	"time"
)

func main() {
	log.Printf("env: %q", os.Environ())
	log.Printf("args: %q", os.Args)

	doc := js.Global().Get("document")
	demo := doc.Call("querySelector", "body #demo")

	if !demo.Truthy() {
		demo = doc.Call("createElement", "h1")
		demo.Call("setAttribute", "id", "demo")
		doc.Get("body").Call("appendChild", demo)
	}
	demo.Set("innerText", fmt.Sprintf("hello from Go, the time is %v", time.Now()))
}
