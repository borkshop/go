// +build !js

package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jcorbin/gorunwasm/handler"
)

var listen = "localhost:8080"

func main() {
	flag.StringVar(&listen, "listen", listen, "http listen address")
	flag.Parse()
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	pkg, err := build.Default.Import("borkshop/automaton", wd, 0)
	if err != nil {
		return err
	}
	srcDir := pkg.Dir

	// TODO better access to the gorunwasm runner script
	gorunwasm, err := build.Default.Import("github.com/jcorbin/gorunwasm", wd, 0)
	if err != nil {
		return err
	}
	http.Handle("/index.js", serveFile(filepath.Join(gorunwasm.Dir, "index.js")))

	http.Handle("/", http.FileServer(http.Dir(srcDir)))
	wh, err := handler.NewWASMHandler(srcDir, ".")
	if err != nil {
		return err
	}

	http.Handle("/main.wasm", wh)
	http.Handle("/wasm_exec.js", serveFile(wh.WASMExec()))

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listen %q failed: %v", listen, err)
	}

	log.Printf("Serving %q on http://%s", srcDir, listen)
	return http.Serve(ln, nil)
}

type serveFile string

func (sf serveFile) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, string(sf))
}
