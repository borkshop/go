// +build !js

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
