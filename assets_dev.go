// +build !js,dev

package main

func init() {
	mux.Handle("/", serveFile("index.html"))
	mux.Handle("/index.js", serveFile("index.js"))
}
