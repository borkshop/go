// +build !js,!dev

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
	indexHandler = staticHandler("index.html", "<!doctype html>\n\n<meta charset=\"utf-8\">\n<title>Go Run</title>\n\n<body>\n\n\t<footer id=\"status\">Loading...</footer>\n\t<script src=\"wasm_exec.js\"></script>\n\t<script src=\"index.js\" data-status-selector=\"#status\"></script>\n\n</body>\n")
	mux.Handle("/index.js", staticHandler("index.js", "global.GoRunner = class {\n\tconstructor(opts) {\n\t\tif (!opts) opts = {};\n\t\tthis.el = opts.el;\n\t\tthis.href = opts.href;\n\t\tthis.data = null;\n\t\tthis.module = null;\n\t\tthis.argv0 = 'wasm';\n\t\tthis.run = null;\n\t\tif (opts.run) {\n\t\t\tthis.run = Array.isArray(opts.run) ? opts.run : [];\n\t\t}\n\t\tthis.load();\n\t}\n\n\tasync load() {\n\t\tlet resp = await fetch(this.href);\n\t\tthis.data = await resp.json();\n\t\tif (document.title === 'Go Run') {\n\t\t\tdocument.title += ': ' + this.data.Package.ImportPath;\n\t\t}\n\t\tif (this.el) {\n\t\t\tthis.el.innerHTML = `Building <tt>${this.data.Package.ImportPath}</tt>...`;\n\t\t}\n\n\t\tconst basename = this.data.Package.Dir.split('/').pop();\n\t\tthis.argv0 = basename + '.wasm';\n\n\t\tresp = await fetch(this.data.Bin);\n\t\tif (/^text\\/plain($|;)/.test(resp.headers.get('Content-Type'))) {\n\t\t\tif (this.el) {\n\t\t\t\tthis.el.innerHTML = `<pre class=\"buildLog\"></pre>`;\n\t\t\t\tthis.el.querySelector('pre').innerText = await resp.text();\n\t\t\t} else {\n\t\t\t\tconsole.error(await resp.text());\n\t\t\t}\n\t\t\treturn;\n\t\t}\n\t\tthis.module = await WebAssembly.compileStreaming(resp);\n\n\t\tif (this.el && !this.run) {\n\t\t\treturn this.interact();\n\t\t}\n\n\t\tlet argv = [this.argv0];\n\t\tif (this.run) {\n\t\t\targv = argv.concat(this.run);\n\t\t}\n\n\t\tif (this.el) {\n\t\t\tthis.el.innerHTML = 'Running...';\n\t\t\tthis.el.style.display = 'none';\n\t\t}\n\n\t\tawait this.run(argv);\n\n\t\tif (this.el) {\n\t\t\tthis.el.style.display = '';\n\t\t\tthis.el.innerHTML = 'Done.';\n\t\t}\n\t}\n\n\tasync interact() {\n\t\tthis.el.innerHTML = `<input class=\"argv\" size=\"40\" title=\"JSON-encoded ARGV\" /><button class=\"run\">Run</button>`;\n\t\tconst runButton = this.el.querySelector('button.run');\n\t\tconst argvInput = this.el.querySelector('input.argv');\n\n\t\targvInput.value = JSON.stringify([this.argv0]);\n\n\t\trunButton.onclick = async () => {\n\t\t\tif (runButton.disabled) return;\n\n\t\t\tconst argv = JSON.parse(argvInput.value);\n\n\t\t\trunButton.disabled = true;\n\t\t\trunButton.innerText = 'Running...';\n\t\t\tthis.el.style.display = 'none';\n\n\t\t\tconsole.clear();\n\t\t\tawait this.run(argv);\n\n\t\t\tthis.el.style.display = '';\n\t\t\trunButton.innerText = 'Run';\n\t\t\trunButton.disabled = false;\n\t\t};\n\t}\n\n\tasync run(argv) {\n\t\tconst go = new Go();\n\t\tgo.argv = argv;\n\t\tconst instance = await WebAssembly.instantiate(this.module, go.importObject);\n\t\tawait go.run(instance);\n\t}\n};\n\n(() => {\n\tconst scr = document.currentScript;\n\tconst href = scr.getAttribute('data-href') || 'build.json';\n\tconst elSel = scr.getAttribute('data-status-selector')\n\tconst runData = scr.getAttribute('data-run') || null;\n\tconst run = runData ? JSON.parse(runData) : null;\n\tconst el = document.querySelector(elSel) || null;\n\tglobal.goRun = new GoRunner({el, run, href});\n})();\n"))
}
