// +build ignore

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"text/template"
)

var tmpl = template.Must(template.New("").Parse(`// +build !js,!dev

package main

import (
	"net/http"
	"strings"
	"time"
)

var staticContentModTime = time.Now()

func staticHandler(name, content string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, name, staticContentModTime, strings.NewReader(content))
	})
}

func init() {
	{{- range . -}}
	mux.Handle("{{ .Path }}", staticHandler("{{ .Name }}", {{ printf "%q" .Content }}))
	{{ end -}}
}
`))

func run() error {
	inr, inw, err := os.Pipe()
	if err != nil {
		return err
	}

	f, err := os.Create("assets_static.go")
	if err != nil {
		return err
	}

	cmd := exec.Command("gofmt")
	cmd.Stdin = inr
	cmd.Stdout = f
	if err := cmd.Start(); err != nil {
		return err
	}
	inr.Close()

	defer f.Close()
	defer log.Printf("wrote %s", f.Name())

	tmpl.Execute(inw, []struct {
		Path    string
		Name    string
		Content string
	}{
		{"/", "index.html", slurp("index.html")},
		{"/index.js", "index.js", slurp("index.js")},
	})
	inw.Close()

	return cmd.Wait()
}

func slurp(name string) string {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
