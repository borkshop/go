async function init() {
	const statusEl = document.querySelector('#status');

	let resp = await fetch("build.json");
	const buildInfo = await resp.json();
	document.title += ': ' + buildInfo.Package.ImportPath;
	statusEl.innerHTML = `Building <tt>${buildInfo.Package.ImportPath}</tt>...`;

	resp = await fetch("main.wasm");

	if (/^text\/plain($|;)/.test(resp.headers.get('Content-Type'))) {
		statusEl.innerHTML = `<pre id="buildLog"></pre>`;
		const log = document.querySelector('#buildLog');
		log.innerText = await resp.text();
		return;
	}

	const module = await WebAssembly.compileStreaming(resp);

	statusEl.innerHTML = `<input id="argv" size="40" title="JSON-encoded ARGV" /><button id="run">Run</button>`;
	const runButton = document.querySelector('#run');
	const argvInput = document.querySelector('#argv');

	const basename = buildInfo.Package.Dir.split('/').pop();
	argvInput.value = JSON.stringify([basename + '.wasm']);

	runButton.onclick = async function() {
		const argv = JSON.parse(argvInput.value);

		runButton.disabled = true;
		runButton.innerText = 'Running...';

		console.clear();
		console.log('Running', buildInfo.Package, 'with args', argv);

		const go = new Go();
		go.argv = argv;
		const instance = await WebAssembly.instantiate(module, go.importObject);
		statusEl.style.display = 'none';
		await go.run(instance);

		statusEl.style.display = '';
		runButton.innerText = 'Run';
		runButton.disabled = false;
	};
}

init();
