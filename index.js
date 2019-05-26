// polyfill
if (!WebAssembly.instantiateStreaming) {
	WebAssembly.instantiateStreaming = async (resp, importObject) => {
		const source = await (await resp).arrayBuffer();
		return await WebAssembly.instantiate(source, importObject);
	};
}

const messageEl = document.querySelector('#status');

async function init() {
	let resp = await fetch("build.json");
	const buildInfo = await resp.json();
	document.title += ': ' + buildInfo.Package.ImportPath;
	messageEl.innerHTML = `Building <tt>${buildInfo.Package.ImportPath}</tt>...`;

	resp = await fetch("main.wasm");

	if (/^text\/plain($|;)/.test(resp.headers.get('Content-Type'))) {
		messageEl.innerHTML = `<pre id="buildLog"></pre>`;
		const log = document.querySelector('#buildLog');
		log.innerText = await resp.text();
		return;
	}

	const go = new Go();
	const res = await WebAssembly.instantiateStreaming(resp, go.importObject);
	const module = res.module;
	let instance = res.instance;

	messageEl.innerHTML = `<input id="argv" size="40" title="JSON-encoded ARGV" /><button id="run">Run</button>`;
	const runButton = document.querySelector('#run');
	const argvInput = document.querySelector('#argv');
	argvInput.value = JSON.stringify(go.argv);

	runButton.onclick = async function() {
		const argv = JSON.parse(argvInput.value);

		runButton.disabled = true;
		runButton.innerText = 'Running...';

		console.clear();
		console.log('Running', buildInfo.Package, 'with args', argv);
		go.argv = argv;
		if (instance == null) {
			instance = await WebAssembly.instantiate(module, go.importObject);
		}
		await go.run(instance);
		instance = null;

		runButton.innerText = 'Run';
		runButton.disabled = false;
	};
}

init();
