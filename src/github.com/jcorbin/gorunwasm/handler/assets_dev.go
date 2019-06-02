// +build dev

package handler

func init() {
	IndexHandler = serveFile("index.html")
	RunHandler = serveFile("index.js")
}
