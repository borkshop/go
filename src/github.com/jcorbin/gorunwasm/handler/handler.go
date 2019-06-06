// Package handler implements a dynamic wasm building http.Handler.
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
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

	wasmPath string
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

// ExecHandler returns an http handler that will serve the appropriate
// wasm_exec.js stub from $GOROOT.
func (wh *WASMHandler) ExecHandler() http.Handler {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	return serveFile(wh.wasmExec)
}

func (wh *WASMHandler) String() string {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	pkg := wh.pkg[entry{wh.srcDir, wh.path}]
	return fmt.Sprintf("WASMHandler %q => %q", pkg.ImportPath, pkg.Dir)
}

func (wh *WASMHandler) packageDir() string {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	return wh.pkg[entry{wh.srcDir, wh.path}].Dir
}

// Close removes any persistent resources.
func (wh *WASMHandler) Close() error {
	return nil
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
	http.ServeFile(w, req, wh.wasmPath)
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
	cmd := exec.Command("go", "build", "-o", wh.wasmPath, importPath)
	cmd.Env = wh.buildEnv()
	cmd.Stderr = &wh.wasmLog
	cmd.Dir = wh.srcDir

	fmt.Fprintf(&wh.wasmLog, "Building %s\n", importPath)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(&wh.wasmLog, "\n%v\n", err)
		return nil
	}

	wh.wasmTime = t0
	wh.wasmOk = true
	return nil
}

func (wh *WASMHandler) buildNeeded() (bool, error) {
	info, err := os.Stat(wh.wasmPath)
	if err == syscall.ENOENT || !wh.wasmOk {
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
	wasmTime := wh.wasmTime
	if ft := info.ModTime(); ft.After(wasmTime) {
		wasmTime = ft
	}
	return mt.After(wasmTime), nil
}

func (wh *WASMHandler) pkgModTime() (time.Time, error) {
	var mtc modTimeChecker
	var ps pkgStack
	ps.add(wh.srcDir, wh.path)
	for !ps.empty() {
		if ent, ok := ps.pop(); ok {
			pkg := wh.pkg[ent]
			mtc.offer(wh.pkgTime[ent])
			mtc.check(pkg.Dir)
			for _, file := range pkg.GoFiles {
				file = filepath.Join(pkg.Dir, file)
				file = filepath.Clean(file)
				mtc.check(file)
			}
			ps.extend(pkg.Dir, pkg.Imports...)
		}
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
	var ps pkgStack
	ps.add(wh.srcDir, wh.path)
	for !ps.empty() {
		if ent, ok := ps.pop(); ok {
			pkg, err := wh.ctxt.Import(ent.path, ent.srcDir, 0)
			if err != nil {
				return fmt.Errorf("failed to import %q in %q: %v", ent.path, ent.srcDir, err)
			}
			wh.pkg[ent] = pkg
			wh.pkgTime[ent] = time.Now()
			ps.extend(pkg.Dir, pkg.Imports...)
		}
	}

	pkg := wh.pkg[entry{wh.srcDir, wh.path}]
	wh.wasmPath = filepath.Join(pkg.Dir, "main.wasm")
	return nil
}

type pkgStack struct {
	seen  map[entry]struct{}
	stack []entry
}

func (ps *pkgStack) empty() bool {
	return len(ps.stack) == 0
}

func (ps *pkgStack) pop() (entry, bool) {
	i := len(ps.stack) - 1
	if i < 0 {
		return entry{}, false
	}
	ent := ps.stack[i]
	ps.stack = ps.stack[:i]
	_, seen := ps.seen[ent]
	if !seen {
		ps.seen[ent] = struct{}{}
	}
	return ent, !seen
}

func (ps *pkgStack) extend(srcDir string, paths ...string) {
	for _, path := range paths {
		ps.add(srcDir, path)
	}
}

func (ps *pkgStack) add(srcDir, path string) {
	if ps.seen == nil {
		ps.seen = make(map[entry]struct{}, 64)
	}
	if ps.stack == nil {
		ps.stack = make([]entry, 0, 64)
	}
	ent := entry{srcDir, path}
	if _, have := ps.seen[ent]; !have {
		ps.stack = append(ps.stack, ent)
	}
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
