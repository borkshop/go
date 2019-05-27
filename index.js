global.GoRunner = class {
	constructor(opts) {
		if (!opts) opts = {};
		this.el = opts.el;
		this.href = opts.href || 'build.json';
		this.data = null;
		this.module = null;
		this.argv0 = 'wasm';
		this.autorun = null;
		if (opts.run) {
			this.autorun = Array.isArray(opts.run) ? opts.run : [];
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

		resp = await fetch("main.wasm");
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

		if (this.el && !this.autorun) {
			return this.interact();
		}

		let argv = [this.argv0];
		if (this.autorun) {
			argv = argv.concat(this.autorun);
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
	const el = document.querySelector('#status');

	global.goRun = new GoRunner({el});
})();
