/*
Package main implements the gorunwasm command, which runs an http server
around a dynamic wasm building handler, along side a DOM-integrated runner
script.

It hosts /wasm_exec.js as shipped with $GOROOT to implement the core Go class
in the browser.

It hosts the wasm handler at /main.wasm.

It hosts a static /index.html wrapper which combines these, or an
http.FileServer around the target package directory if it contains an
index.html.

The runner script supports passing environment variables to the built Go
wasm/js program:

	<!doctype html>

	<title>Go WASM Hello World</title>

	<body>

		<div>
			<label for="who">Who are you?</label>
			<input id="who" type="text" size="20">
		</div>

		<div id="output">...</div>

		<script src="wasm_exec.js"></script>
		<script src="index.js" data-input="#who" data-output="#output"></script>

	</body>

Which can then be used along side a corresponding main.go:

	// +build js

	package main

	import (
		"fmt"
		"os"
		"syscall/js"
	)

	var document = js.Global().Get("document")

	func main() {
		input := document.Call("querySelector", os.Getenv("input"))
		output := document.Call("querySelector", os.Getenv("output"))
		input.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			val := input.Get("value")
			output.Set("innerText", fmt.Sprintf("Hello %q", val))
			return nil
		}))
		select {}
	}

The runner script also supports setting data-argv0 and data-args attributes to
pass command line arguments. However environment variables are easier and less
error prone to use, since they are simple key/value passing, where-as the args
value must be a JSON encoded array of strings.

The runner script displays any text/plain content by replacing all body content
with it. This aligns with WASMHandler redirecting to the build log on build
failure.

*/
package main
