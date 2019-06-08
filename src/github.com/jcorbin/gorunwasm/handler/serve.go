package handler

import (
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
	if srcDir == "" {
		var err error
		srcDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}

		pkg, err := build.Default.Import(path, srcDir, build.FindOnly)
		if err != nil {
			return nil, err
		}
		srcDir, path = pkg.Dir, "."
	}

	wh, err := NewWASMHandler(srcDir, path)
	if err == nil {
		wh.Mount(prefix, http.DefaultServeMux)
	}
	return wh, err
}

// UploadHandler implements an http.Handler that will accept and save POST-ed
// entities into a directory.
type UploadHandler string

func (uh UploadHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseMultipartForm(64 * 1024 * 1024 * 1024); err == nil {
		for name, fls := range req.MultipartForm.File {
			fl := fls[0]
			f, err := fl.Open()
			if err == nil {
				err = uh.accept(name, f)
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		return
	}

	name := req.FormValue("name")
	if name == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}
	if err := uh.accept(name, req.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (uh UploadHandler) accept(name string, in io.ReadCloser) error {
	outPath := string(uh)
	for _, part := range strings.Split(name, "/") {
		outPath = filepath.Join(outPath, part)
	}

	out, err := createOrMkdir(outPath)

	log.Printf("receiving upload to %q", out.Name())
	defer func() {
		if err != nil {
			log.Printf("upload to %q failed: %v", out.Name(), err)
		} else {
			log.Printf("upload to %q done", out.Name())
		}
	}()

	var buf [32 * 1024]byte
	_, err = io.CopyBuffer(out, in, buf[:])

	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if cerr := in.Close(); err == nil {
		err = cerr
	}

	return err
}

func createOrMkdir(name string) (*os.File, error) {
	out, err := os.Create(name)
	if err != nil {
		if unwrapOSError(err) == syscall.ENOENT {
			dir := filepath.Dir(name)
			err = os.MkdirAll(dir, os.ModePerm)
			if err == nil {
				out, err = os.Create(name)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func unwrapOSError(err error) error {
	for {
		switch val := err.(type) {
		case *os.PathError:
			err = val.Err
		case *os.LinkError:
			err = val.Err
		case *os.SyscallError:
			err = val.Err
		default:
			return err
		}
	}
}
