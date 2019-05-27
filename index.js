global.GoRunner = class {
	constructor(opts) {
		if (!opts) opts = {};
		this.el = opts.el;
		this.href = opts.href;
		this.data = null;
		this.module = null;
		this.argv0 = 'wasm';
		this.run = null;
		if (opts.run) {
			this.run = Array.isArray(opts.run) ? opts.run : [];
		}
		this.load();
	}

	async load() {
		let resp = await fetch(this.href);
		this.data = await resp.json();
		if (document.title === 'Go Run') {
			document.title += ': ' + this.data.Package.ImportPath;
		}
		if (this.el) {
			this.el.innerHTML = `Building <tt>${this.data.Package.ImportPath}</tt>...`;
		}

		const basename = this.data.Package.Dir.split('/').pop();
		this.argv0 = basename + '.wasm';

		resp = await fetch(this.data.Bin);
		if (/^text\/plain($|;)/.test(resp.headers.get('Content-Type'))) {
			if (this.el) {
				this.el.innerHTML = `<pre class="buildLog"></pre>`;
				this.el.querySelector('pre').innerText = await resp.text();
			} else {
				console.error(await resp.text());
			}
			return;
		}
		this.module = await WebAssembly.compileStreaming(resp);

		if (this.el && !this.run) {
			return this.interact();
		}

		let argv = [this.argv0];
		if (this.run) {
			argv = argv.concat(this.run);
		}

		if (this.el) {
			this.el.innerHTML = 'Running...';
			this.el.style.display = 'none';
		}

		await this.run(argv);

		if (this.el) {
			this.el.style.display = '';
			this.el.innerHTML = 'Done.';
		}
	}

	async interact() {
		this.el.innerHTML = `<input class="argv" size="40" title="JSON-encoded ARGV" /><button class="run">Run</button>`;
		const runButton = this.el.querySelector('button.run');
		const argvInput = this.el.querySelector('input.argv');

		argvInput.value = JSON.stringify([this.argv0]);

		runButton.onclick = async () => {
			if (runButton.disabled) return;

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
	const scr = document.currentScript;
	const href = scr.getAttribute('data-href') || 'build.json';
	const elSel = scr.getAttribute('data-status-selector')
	const runData = scr.getAttribute('data-run') || null;
	const run = runData ? JSON.parse(runData) : null;
	const el = document.querySelector(elSel) || null;
	global.goRun = new GoRunner({el, run, href});
})();
