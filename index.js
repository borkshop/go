global.GoRunner = class {
	constructor(opts) {
		if (!opts) opts = {};
		this.el = opts.el;
		this.href = opts.href || 'build.json';
		this.data = null;
		this.module = null;
		this.argv0 = 'wasm';
		this.load();
	}

	async load() {
		let resp = await fetch(this.href);
		this.data = await resp.json();
		document.title += ': ' + this.data.Package.ImportPath;
		this.el.innerHTML = `Building <tt>${this.data.Package.ImportPath}</tt>...`;

		const basename = this.data.Package.Dir.split('/').pop();
		this.argv0 = basename + '.wasm';

		resp = await fetch("main.wasm");
		if (/^text\/plain($|;)/.test(resp.headers.get('Content-Type'))) {
			this.el.innerHTML = `<pre id="buildLog"></pre>`;
			const log = document.querySelector('#buildLog');
			log.innerText = await resp.text();
			return;
		}
		this.module = await WebAssembly.compileStreaming(resp);

		return this.interact();
	}

	async interact() {
		this.el.innerHTML = `<input id="argv" size="40" title="JSON-encoded ARGV" /><button id="run">Run</button>`;
		const runButton = document.querySelector('#run');
		const argvInput = document.querySelector('#argv');

		argvInput.value = JSON.stringify([this.argv0]);

		runButton.onclick = async () => {
			const argv = JSON.parse(argvInput.value);

			runButton.disabled = true;
			runButton.innerText = 'Running...';
			this.el.style.display = 'none';

			console.clear();
			await this.run(argv);

			this.el.style.display = '';
			runButton.innerText = 'Run';
			runButton.disabled = false;
		};
	}

	async run(argv) {
		const go = new Go();
		go.argv = argv;
		const instance = await WebAssembly.instantiate(this.module, go.importObject);
		await go.run(instance);
	}
};

(() => {
	const el = document.querySelector('#status');

	global.goRun = new GoRunner({el});
})();
