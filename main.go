/*
package main implements the gorunwasm command, which runs an http server
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

*/
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jcorbin/gorunwasm/handler"
)

//go:generate go run assets_build.go

var (
	mux          = http.NewServeMux()
	indexHandler http.Handler
)

func run() error {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen", "localhost:0", "listen address for http server")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}
	srcDir := wd

	flag.Parse()
	args := flag.Args()

	path := "."
	if len(args) > 0 {
		path = args[0]
		if filepath.IsAbs(path) {
			srcDir = args[0]
			path = "."
		}
	}

	wh, err := handler.NewWASMHandler(srcDir, path)
	if err != nil {
		return err
	}
	defer wh.Close()

	mux.Handle("/wasm_exec.js", serveFile(wh.WASMExec()))
	mux.Handle("/main.wasm", wh)

	pkgDir := wh.PackageDir()
	if _, err := os.Stat(filepath.Join(pkgDir, "index.html")); err == nil {
		log.Printf("Serving http files from %q", pkgDir)
		mux.Handle("/", http.FileServer(http.Dir(pkgDir)))
	} else {
		log.Printf("Providing default index handler")
		mux.Handle("/", indexHandler)
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %q failed: %v", listenAddr, err)
	}

	log.Printf("listening on http://%v", ln.Addr())

	return http.Serve(ln, mux)
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

type serveFile string

func (sf serveFile) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, string(sf))
}
