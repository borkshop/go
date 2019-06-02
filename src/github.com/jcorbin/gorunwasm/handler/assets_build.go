// +build ignore

package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"text/template"
)

var tmpl = template.Must(template.New("").Parse(`// +build !dev

package handler

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
	{{- range . }}
	{{ .Handler }} = staticHandler("{{ .Name }}", {{ printf "%q" .Content }})
	{{- end -}}
}
`))

type servedFile struct {
	Handler string
	Name    string
	Content string
}

var serveFilePattern = regexp.MustCompile(`(\w+) = serveFile\("(.+?)"\)`)

func readServedFiles(r io.Reader) ([]servedFile, error) {
	var servedFiles []servedFile
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if match := serveFilePattern.FindSubmatch(sc.Bytes()); len(match) > 0 {
			handler := string(match[1])
			name := string(match[2])
			servedFiles = append(servedFiles, servedFile{handler, name, slurp(name)})
		} else {

			log.Printf("WUT %q", sc.Bytes())
		}
	}
	return servedFiles, sc.Err()
}

func run() error {
	f, err := os.Open("assets_dev.go")
	if err != nil {
		return err
	}
	defer f.Close()

	servedFiles, err := readServedFiles(f)
	if err != nil {
		return err
	}

	f, err = os.Create("assets_static.go")
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, servedFiles); err != nil {
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
