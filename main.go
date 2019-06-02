package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jcorbin/gorunwasm/handler"
)

var (
	listen = "localhost:0"
	gen    bool // server:skip

	srcDir = ""
	path   = "." // server:skip
)

func main() {
	flag.StringVar(&listen, "listen", listen, "listen address for http server")
	flag.BoolVar(&gen, "gen", false, "generate server main code rather than run a generic server")

	flag.Parse()

	if args := flag.Args(); len(args) > 0 {
		path = args[0]
		if filepath.IsAbs(path) {
			srcDir = args[0]
			path = "."
		}
	}

	var err error
	if gen {
		err = genServer()
	} else {
		err = serve()
	}
	if err != nil {
		log.Fatalln(err)
	}
}

func genServer() error {
	// resolve target package
	if srcDir == "" {
		var err error
		srcDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	pkg, err := build.Default.Import(path, srcDir, 0)
	if err != nil {
		return err
	}

	out, err := os.Create(filepath.Join(pkg.Dir, "server.go"))
	if err != nil {
		return err
	}
	defer out.Close()

	// execute template to stdout through gofmt
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd := exec.Command("gofmt")
	cmd.Stdin = r
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	_ = r.Close()
	if err == nil {
		err = tmpl.Execute(w, pkg)
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if werr := cmd.Wait(); err == nil {
			err = werr
		}
	}
	if err == nil {
		log.Printf("generated %v", out.Name())
	}
	return err
}

func serve() error {
	wh, err := handler.Handle("", srcDir, path)
	if err != nil {
		return err
	}
	defer wh.Close()

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listen %q failed: %v", listen, err)
	}

	log.Printf("listening on http://%v", ln.Addr())
	log.Printf("Serving %v on http://%s", wh, ln.Addr())

	return http.Serve(ln, nil)
}
