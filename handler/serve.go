package handler

import (
	"net/http"
	"os"
	"path/filepath"
)

//go:generate go run assets_build.go

var (
	// IndexHandler provides a sensible default index.html
	IndexHandler http.Handler // index.html

	// RunHandler provides a wrapper script that eases integrating wasm_exec.js
	// and a compiled wasm endpoint.
	RunHandler http.Handler // index.js
)

type serveFile string

func (sf serveFile) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, string(sf))
}

// IndexHandler returns an http.Handler either backed by the package's
// directory if it contains an index.html file, or a default one otherwise.
func (wh *WASMHandler) IndexHandler() http.Handler {
	pkgDir := wh.packageDir()
	if _, err := os.Stat(filepath.Join(pkgDir, "index.html")); err == nil {
		return http.FileServer(http.Dir(pkgDir))
	}
	return IndexHandler
}

// Mount mounts the IndexHandler() at /, the RunHandler at /index.js, the
// ExecHandler() at /wasm_exec.js, and finally the WASMHandler itself at
// /main.wasm.
func (wh *WASMHandler) Mount(prefix string, mux *http.ServeMux) {
	mux.Handle(prefix+"/", wh.IndexHandler())
	mux.Handle(prefix+"/index.js", RunHandler)
	mux.Handle(prefix+"/wasm_exec.js", wh.ExecHandler())
	mux.Handle(prefix+"/main.wasm", wh)
}

// Handle mounts a new WASMHandler at the given prefix onto the
// http.DefaultServeMux. The caller should defer a call WASMHandler.Close() to
// ensure temporary file deletion.
func Handle(prefix, srcDir, path string) (*WASMHandler, error) {
	wh, err := NewWASMHandler(srcDir, path)
	if err == nil {
		wh.Mount(prefix, http.DefaultServeMux)
	}
	return wh, err
}
