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

	wh, err := handler.Handle("", srcDir, path)
	if err != nil {
		return err
	}
	defer wh.Close()

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %q failed: %v", listenAddr, err)
	}

	log.Printf("listening on http://%v", ln.Addr())

	return http.Serve(ln, nil)
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
