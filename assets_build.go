// +build ignore

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"text/template"
)

var tmpl = template.Must(template.New("").Parse(`// +build !dev

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
	{{- if eq .Path "/" -}}
	indexHandler = staticHandler("{{ .Name }}", {{ printf "%q" .Content }})
	{{ else -}}
	mux.Handle("{{ .Path }}", staticHandler("{{ .Name }}", {{ printf "%q" .Content }}))
	{{ end -}}
	{{- end -}}
}
`))

func run() error {
	f, err := os.Create("assets_static.go")
	if err != nil {
		return err
	}

	defer f.Close()

	if err := tmpl.Execute(f, []struct {
		Path    string
		Name    string
		Content string
	}{
		{"/", "index.html", slurp("index.html")},
		{"/index.js", "index.js", slurp("index.js")},
	}); err != nil {
		return err
	}

	log.Printf("wrote %s", f.Name())

	cmd := exec.Command("gofmt", "-w", "assets_static.go")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	log.Printf("formatted %s", f.Name())
	return nil
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
