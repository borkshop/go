// +build !js,dev

package main

func init() {
	indexHandler = serveFile("index.html")
	mux.Handle("/index.js", serveFile("index.js"))
}
