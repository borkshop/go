// +build !js

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:generate go run assets_build.go

var (
	mux = http.NewServeMux()

	srcDir string

	buildContext     = build.Default
	buildPackage     build.Package
	buildPackageTime time.Time

	builtWASMMutex sync.RWMutex
	builtWASM      *os.File
	builtWASMOk    bool
	builtWASMTime  time.Time
	builtWASMLog   bytes.Buffer

	wasmExec = filepath.Join(buildContext.GOROOT, "misc", "wasm", "wasm_exec.js")

	indexHandler http.Handler
)

func init() {
	buildContext.GOARCH = "wasm"
	buildContext.GOOS = "js"
	mux.Handle("/wasm_exec.js", serveFile(wasmExec))
	mux.HandleFunc("/build.json", handleBuildJSON)
	mux.HandleFunc("/build.log", handleBuildLog)
	mux.HandleFunc("/main.wasm", handleMainWasm)
}

func handleBuildJSON(w http.ResponseWriter, req *http.Request) {
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
		Bin     string
		Context builtContext
		Package build.Package
	}{"main.wasm", builtContext{
		buildContext.GOARCH,
		buildContext.GOOS,
		buildContext.GOROOT,
		buildContext.GOPATH,
		buildContext.CgoEnabled,
		buildContext.UseAllFiles,
		buildContext.Compiler,
		buildContext.BuildTags,
		buildContext.ReleaseTags,
		buildContext.InstallSuffix,
	}, buildPackage}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal json: %v", err), http.StatusInternalServerError)
	}
}

func handleBuildLog(w http.ResponseWriter, req *http.Request) {
	builtWASMMutex.RLock()
	defer builtWASMMutex.RUnlock()
	http.ServeContent(w, req, "build.log", builtWASMTime, bytes.NewReader(builtWASMLog.Bytes()))
}

func handleMainWasm(w http.ResponseWriter, req *http.Request) {
	if need, err := wasmBuildNeeded(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if need {
		if err := buildWASM(); err != nil {
			http.Error(w,
				fmt.Sprintf("failed to build wasm: %v", err),
				http.StatusInternalServerError)
			return
		}
	}

	builtWASMMutex.RLock()
	defer builtWASMMutex.RUnlock()
	if !builtWASMOk {
		http.Redirect(w, req, "/build.log", http.StatusSeeOther)
	} else {
		http.ServeContent(w, req, "main.wasm", builtWASMTime, builtWASM)
	}
}

func wasmBuildNeeded() (bool, error) {
	builtWASMMutex.RLock()
	defer builtWASMMutex.RUnlock()
	if builtWASM != nil {
		if _, err := builtWASM.Seek(0, os.SEEK_SET); err != nil {
			return true, nil
		}
	}
	if builtWASM == nil || !builtWASMOk {
		return true, nil
	}
	mt, err := buildPackageModTime()
	if err != nil {
		return false, fmt.Errorf("failed to get build package mod time: %v", err)
	}
	return mt.After(builtWASMTime), nil
}

func buildPackageModTime() (time.Time, error) {
	t := buildPackageTime
	for _, path := range buildPackagePaths() {
		info, err := os.Stat(path)
		if err != nil {
			err = lookupPackage("")
			return buildPackageTime, err
		}
		if it := info.ModTime(); it.After(t) {
			t = it
		}
	}
	return t, nil
}

func buildPackagePaths() []string {
	paths := make([]string, 1, len(buildPackage.GoFiles)+1)
	paths[0] = buildPackage.Dir
	paths = append(paths, buildPackage.GoFiles...)
	return paths
}

func buildWASM() error {
	builtWASMMutex.Lock()
	defer builtWASMMutex.Unlock()

	if builtWASM != nil {
		if _, err := builtWASM.Seek(0, os.SEEK_SET); err != nil {
			removeBuiltWasm()
		}
	}
	if builtWASM == nil {
		if err := openBuiltWasm(); err != nil {
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
		_, err := io.Copy(builtWASM, pr)
		if closeErr := pr.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			_, err = builtWASM.Seek(0, os.SEEK_SET)
		}
		copyChan <- err
	}()

	t0 := time.Now()
	defer func() {
		t1 := time.Now()
		fmt.Fprintf(&builtWASMLog, "\nBuild Took %v\n", t1.Sub(t0))
	}()

	builtWASMTime = time.Time{}
	builtWASMOk = false
	builtWASMLog.Reset()
	builtWASMLog.Grow(64 * 1024)

	cmd := exec.Command("go", "build", "-o", "/dev/stdout", buildPackage.ImportPath)
	cmd.Env = buildEnv()
	cmd.Stdout = pw
	cmd.Stderr = &builtWASMLog
	cmd.Dir = srcDir

	fmt.Fprintf(&builtWASMLog, "Building %s\n", buildPackage.ImportPath)

	err = cmd.Start()
	_ = pw.Close()
	if err == nil {
		err = cmd.Wait()
	}

	if err != nil {
		fmt.Fprintf(&builtWASMLog, "\n%v\n", err)
	}

	if err != nil {
		return nil
	}

	if copyErr := <-copyChan; copyErr != nil {
		removeBuiltWasm()
		return fmt.Errorf("build output copy failed: %v", err)
	}

	builtWASMTime = time.Now()
	builtWASMOk = true
	return nil
}

func buildEnv() []string {
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
		{"GOARCH", buildContext.GOARCH},
		{"GOOS", buildContext.GOOS},
		{"GOROOT", buildContext.GOROOT},
		{"GOPATH", buildContext.GOPATH},
	} {
		if pair[1] != "" {
			env = append(env, fmt.Sprintf("%s=%s", pair[0], pair[1]))
		}
	}
	return env
}

func run() error {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen", "localhost:0", "listen address for http server")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}
	srcDir = wd

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
	if err := lookupPackage(path); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(buildPackage.Dir, "index.html")); err == nil {
		log.Printf("Serving http files from %q", buildPackage.Dir)
		mux.Handle("/", http.FileServer(http.Dir(buildPackage.Dir)))
	} else {
		log.Printf("Providing default index handler")
		mux.Handle("/", indexHandler)
	}

	defer removeBuiltWasm()

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %q failed: %v", listenAddr, err)
	}

	log.Printf("listening on http://%v", ln.Addr())

	return http.Serve(ln, mux)
}

func lookupPackage(path string) error {
	if path == "" {
		path = buildPackage.ImportPath
	}
	if path == "" {
		return errors.New("no build package path set")
	}
	pkg, err := buildContext.Import(path, srcDir, 0)
	if err != nil {
		return fmt.Errorf("failed to import %q: %v", path, err)
	}
	buildPackage = *pkg
	buildPackageTime = time.Now()
	return nil
}

func openBuiltWasm() error {
	removeBuiltWasm()
	f, err := ioutil.TempFile("", "main.wasm")
	if err != nil {
		return err
	}
	builtWASM = f
	return nil
}

func removeBuiltWasm() {
	if builtWASM != nil {
		_ = os.Remove(builtWASM.Name())
		_ = builtWASM.Close()
		builtWASM = nil
	}
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
