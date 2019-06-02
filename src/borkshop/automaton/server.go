// +build !js

//go:generate go run github.com/jcorbin/gorunwasm -gen borkshop/automaton

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/jcorbin/gorunwasm/handler"
)

var (
	listen = "localhost:0"

	srcDir = ""
	path   = "borkshop/automaton"
)

func main() {
	flag.StringVar(&listen, "listen", listen, "listen address for http server")
	flag.Parse()
	log.Fatalln(serve())
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
