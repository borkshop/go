// Package handler implements a dynamic wasm building http.Handler.
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WASMHandler implements an http.Handler that serves a dynamically built wasm
// binary from a Go "main" package.
//
// The target package must be a normal main package with a func main() entry
// point and should contain a js build tag. See the package examples for detail.
type WASMHandler struct {
	mu sync.RWMutex

	srcDir string
	ctxt   build.Context
	path   string

	pkg      map[entry]*build.Package
	pkgTime  map[entry]time.Time
	wasmExec string // class Go implemented by $GOROOT/misc/wasm_exec.js

	wasm     *os.File
	wasmOk   bool
	wasmTime time.Time
	wasmLog  bytes.Buffer
}

type entry struct {
	srcDir, path string
}

// NewWASMHandler creates a WASMHandler for a given package path and source
// directory.
func NewWASMHandler(srcDir, path string) (*WASMHandler, error) {
	var wh WASMHandler

	wh.srcDir = srcDir
	wh.ctxt = build.Default
	wh.ctxt.GOARCH = "wasm"
	wh.ctxt.GOOS = "js"
	wh.path = path
	wh.wasmExec = filepath.Join(wh.ctxt.GOROOT, "misc", "wasm", "wasm_exec.js")
	wh.pkg = make(map[entry]*build.Package)
	wh.pkgTime = make(map[entry]time.Time)
	if err := wh.refreshPackage(); err != nil {
		return nil, err
	}

	return &wh, nil
}

// WASMExec returns the path to the runtime wasm_exec.js stub.
func (wh *WASMHandler) WASMExec() string {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	return wh.wasmExec
}

// PackageDir returns the directory path to the main package being built as a
// wasm binary.
func (wh *WASMHandler) PackageDir() string {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	return wh.pkg[entry{wh.srcDir, wh.path}].Dir
}

// Close removes any temporary built wasm binary.
func (wh *WASMHandler) Close() error {
	wh.mu.Lock()
	defer wh.mu.Unlock()
	return wh.removeWasm()
}

// ServeHTTP dispatches the request dynamically.
//
// It serves a text build log if the "log" form value is set.
//
// It serves a json build config if the "build" form value is set.
//
// It builds a wasm binary if none has been built before or if the "force" form value is set.
//
// It serves the built wasm binary, or redirects to the build log if the build fails.
func (wh *WASMHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if _, logSet := req.Form["log"]; logSet {
		wh.serveLog(w, req)
		return
	}
	if _, buildSet := req.Form["build"]; buildSet {
		wh.serveJSON(w, req)
		return
	}
	wh.serveWASM(w, req)
}

func (wh *WASMHandler) serveWASM(w http.ResponseWriter, req *http.Request) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	_, forceSet := req.Form["force"]

	doBuild := forceSet
	if !doBuild {
		if need, err := wh.buildNeeded(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			doBuild = need
		}
	}
	if doBuild {
		if err := wh.build(); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to build wasm: %v", err),
				http.StatusInternalServerError)
			return
		}
	}

	if !wh.wasmOk {
		http.Redirect(w, req, req.URL.Path+"?log", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, req, "main.wasm", wh.wasmTime, wh.wasm)
}

func (wh *WASMHandler) serveJSON(w http.ResponseWriter, req *http.Request) {
	wh.mu.RLock()
	defer wh.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	type builtContext struct {
		GOARCH        string
		GOOS          string
		GOROOT        string
		GOPATH        string
		CgoEnabled    bool
		UseAllFiles   bool
		Compiler      string
		BuildTags     []string
		ReleaseTags   []string
		InstallSuffix string
	}
	if err := json.NewEncoder(w).Encode(struct {
		Context builtContext
		Package *build.Package
	}{builtContext{
		wh.ctxt.GOARCH,
		wh.ctxt.GOOS,
		wh.ctxt.GOROOT,
		wh.ctxt.GOPATH,
		wh.ctxt.CgoEnabled,
		wh.ctxt.UseAllFiles,
		wh.ctxt.Compiler,
		wh.ctxt.BuildTags,
		wh.ctxt.ReleaseTags,
		wh.ctxt.InstallSuffix,
	}, wh.pkg[entry{wh.srcDir, wh.path}]}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal json: %v", err), http.StatusInternalServerError)
	}
}

func (wh *WASMHandler) serveLog(w http.ResponseWriter, req *http.Request) {
	wh.mu.RLock()
	defer wh.mu.RUnlock()

	http.ServeContent(w, req, "build.log", wh.wasmTime, bytes.NewReader(wh.wasmLog.Bytes()))
}

func (wh *WASMHandler) build() error {
	if wh.wasm != nil {
		if _, err := wh.wasm.Seek(0, os.SEEK_SET); err != nil {
			wh.removeWasm()
		}
	}
	if wh.wasm == nil {
		if err := wh.openWasm(); err != nil {
			return fmt.Errorf("unable to create temporary file: %v", err)
		}
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to pipe: %v", err)
	}
	copyChan := make(chan error, 1)
	go func() {
		defer close(copyChan)
		_, err := io.Copy(wh.wasm, pr)
		if closeErr := pr.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			_, err = wh.wasm.Seek(0, os.SEEK_SET)
		}
		copyChan <- err
	}()

	t0 := time.Now()
	defer func() {
		t1 := time.Now()
		fmt.Fprintf(&wh.wasmLog, "\nBuild Took %v\n", t1.Sub(t0))
	}()

	wh.wasmTime = time.Time{}
	wh.wasmOk = false
	wh.wasmLog.Reset()
	wh.wasmLog.Grow(64 * 1024)

	importPath := wh.pkg[entry{wh.srcDir, wh.path}].ImportPath
	cmd := exec.Command("go", "build", "-o", "/dev/stdout", importPath)
	cmd.Env = wh.buildEnv()
	cmd.Stdout = pw
	cmd.Stderr = &wh.wasmLog
	cmd.Dir = wh.srcDir

	fmt.Fprintf(&wh.wasmLog, "Building %s\n", importPath)

	err = cmd.Start()
	_ = pw.Close()
	if err == nil {
		err = cmd.Wait()
	}

	if err != nil {
		fmt.Fprintf(&wh.wasmLog, "\n%v\n", err)
	}

	if err != nil {
		return nil
	}

	if copyErr := <-copyChan; copyErr != nil {
		wh.removeWasm()
		return fmt.Errorf("build output copy failed: %v", err)
	}

	wh.wasmTime = time.Now()
	wh.wasmOk = true
	return nil
}

func (wh *WASMHandler) buildNeeded() (bool, error) {
	if wh.wasm != nil {
		if _, err := wh.wasm.Seek(0, os.SEEK_SET); err != nil {
			return true, nil
		}
	}
	if wh.wasm == nil || !wh.wasmOk {
		return true, nil
	}
	mt, err := wh.pkgModTime()
	if err != nil {
		err = wh.refreshPackage()
		if err == nil {
			mt, err = wh.pkgModTime()
		}
		if err != nil {
			return false, fmt.Errorf("failed to get build package mod time: %v", err)
		}
	}
	return mt.After(wh.wasmTime), nil
}

func (wh *WASMHandler) pkgModTime() (time.Time, error) {
	var mtc modTimeChecker
	ent := entry{wh.srcDir, wh.path}
	pkg := wh.pkg[ent]
	mtc.offer(wh.pkgTime[ent])
	mtc.check(pkg.Dir)
	for _, path := range pkg.GoFiles {
		mtc.check(path)
	}
	return mtc.t, mtc.err
}

func (wh *WASMHandler) buildEnv() []string {
	osEnv := os.Environ()
	env := make([]string, 0, len(osEnv)+4)
	// TODO should we instead just use a whitelist?
	for _, s := range osEnv {
		// skip env keys that contain escape sequences
		if !strings.ContainsRune(s, 0x1b) {
			env = append(env, s)
		}
	}
	for _, pair := range [][2]string{
		{"GOARCH", wh.ctxt.GOARCH},
		{"GOOS", wh.ctxt.GOOS},
		{"GOROOT", wh.ctxt.GOROOT},
		{"GOPATH", wh.ctxt.GOPATH},
	} {
		if pair[1] != "" {
			env = append(env, fmt.Sprintf("%s=%s", pair[0], pair[1]))
		}
	}
	return env
}

func (wh *WASMHandler) refreshPackage() error {
	if wh.path == "" {
		return errors.New("no package path set")
	}
	ent := entry{wh.srcDir, wh.path}
	pkg, err := wh.ctxt.Import(ent.path, ent.srcDir, 0)
	if err != nil {
		return fmt.Errorf("failed to import %q: %v", wh.path, err)
	}
	wh.pkg[ent] = pkg
	wh.pkgTime[ent] = time.Now()
	return nil
}

func (wh *WASMHandler) openWasm() error {
	wh.removeWasm()
	f, err := ioutil.TempFile("", "main.wasm")
	if err != nil {
		return err
	}
	wh.wasm = f
	return nil
}

func (wh *WASMHandler) removeWasm() error {
	if wh.wasm == nil {
		return nil
	}
	err := os.Remove(wh.wasm.Name())
	if cerr := wh.wasm.Close(); err == nil {
		err = cerr
	}
	wh.wasm = nil
	return err
}

type modTimeChecker struct {
	t   time.Time
	err error
}

func (mtc *modTimeChecker) offer(t time.Time) {
	if mtc.t.IsZero() || t.After(mtc.t) {
		mtc.t = t
	}
}

func (mtc *modTimeChecker) check(paths ...string) {
	for _, path := range paths {
		if mtc.err != nil {
			return
		}
		info, err := os.Stat(path)
		if err != nil {
			mtc.err = err
			return
		}
		mtc.offer(info.ModTime())
	}
}
