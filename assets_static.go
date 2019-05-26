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
	mux.Handle("/", staticHandler("index.html", "<!doctype html>\n\n<meta charset=\"utf-8\">\n<title>Go Run</title>\n\n<script src=\"wasm_exec.js\"></script>\n\n<body>\n\n\t<footer id=\"status\">Loading...</footer>\n\t<script src=\"index.js\"></script>\n\n</body>\n"))
	mux.Handle("/index.js", staticHandler("index.js", "// polyfill\nif (!WebAssembly.instantiateStreaming) {\n\tWebAssembly.instantiateStreaming = async (resp, importObject) => {\n\t\tconst source = await (await resp).arrayBuffer();\n\t\treturn await WebAssembly.instantiate(source, importObject);\n\t};\n}\n\nconst messageEl = document.querySelector('#status');\n\nasync function init() {\n\tlet resp = await fetch(\"build.json\");\n\tconst buildInfo = await resp.json();\n\tdocument.title += ': ' + buildInfo.Package.ImportPath;\n\tmessageEl.innerHTML = `Building <tt>${buildInfo.Package.ImportPath}</tt>...`;\n\n\tresp = await fetch(\"main.wasm\");\n\n\tif (/^text\\/plain($|;)/.test(resp.headers.get('Content-Type'))) {\n\t\tmessageEl.innerHTML = `<pre id=\"buildLog\"></pre>`;\n\t\tconst log = document.querySelector('#buildLog');\n\t\tlog.innerText = await resp.text();\n\t\treturn;\n\t}\n\n\tconst go = new Go();\n\tconst res = await WebAssembly.instantiateStreaming(resp, go.importObject);\n\tconst module = res.module;\n\tlet instance = res.instance;\n\n\tmessageEl.innerHTML = `<input id=\"argv\" size=\"40\" title=\"JSON-encoded ARGV\" /><button id=\"run\">Run</button>`;\n\tconst runButton = document.querySelector('#run');\n\tconst argvInput = document.querySelector('#argv');\n\targvInput.value = JSON.stringify(go.argv);\n\n\trunButton.onclick = async function() {\n\t\tconst argv = JSON.parse(argvInput.value);\n\n\t\trunButton.disabled = true;\n\t\trunButton.innerText = 'Running...';\n\n\t\tconsole.clear();\n\t\tconsole.log('Running', buildInfo.Package, 'with args', argv);\n\t\tgo.argv = argv;\n\t\tif (instance == null) {\n\t\t\tinstance = await WebAssembly.instantiate(module, go.importObject);\n\t\t}\n\t\tawait go.run(instance);\n\t\tinstance = null;\n\n\t\trunButton.innerText = 'Run';\n\t\trunButton.disabled = false;\n\t};\n}\n\ninit();\n"))
}
