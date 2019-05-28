global.GoRunner = class {
	// parseConfigData from an element's data-* attributes:
	// - data-href optional URL to fetch the WASM binary, defaults to main.wasm.
	// - data-status-selector may provide a dom query selector for a displaying
	//   build errors and (re)running the Go main.
	// - data-args may provide a JSON-encoded argument array to pass to the Go program.
	// - data-argv0 overrides the program name (argv0) that the Go program is
	//   invoked under; defaults to "package_name.wasm"
	// - any other data-* keys are passed as environment variables to the Go program.
	static parseConfigData(el) {
		const cfg = {
			el: null,
			href: 'main.wasm',
			argv0: null,
			args: null,
			env: {},
		};
		for (let i = 0; i < el.attributes.length; i++) {
			const {nodeName, nodeValue} = el.attributes[i];
			const dataMatch = /^data-(.+)/.exec(nodeName);
			if (!dataMatch) continue;
			const name = dataMatch[1];
			switch (name) {
				case 'href':
					cfg.href = nodeValue;
					break;
				case 'status-selector':
					cfg.el = document.querySelector(nodeValue);
					break;
				case 'argv0':
					cfg.argv0 = nodeValue;
					break;
				case 'args':
					cfg.args = JSON.parse(nodeValue);
					if (!Array.isArray(cfg.args)) throw new Error('data-args must be an array');
					break;
				default:
					cfg.env[name] = nodeValue;
					break;
			}
		}
		return cfg;
	}

	constructor(cfg) {
		this.el = cfg.el;
		this.href = cfg.href;
		this.args = cfg.args;
		this.env = cfg.env;
		this.argv0 = cfg.argv0;
		this.module = null;
		this.load();
	}

	async load() {
		const parseContentType = (resp) => {
			const match = /^([^;]+)/.exec(resp.headers.get('Content-Type'));
			return match ? match[1] : '';
		};

		const setTitle = (title) => {
			if (document.title === 'Go Run') {
				document.title += ': ' + title;
			}
		};

		let resp = await fetch(this.href);
		if (parseContentType(resp) === 'application/json') {
			const data = await resp.json();
			setTitle(data.Package.ImportPath);
			if (this.el) {
				this.el.innerHTML = `Building <tt>${data.Package.ImportPath}</tt>...`;
			}
			const basename = data.Package.Dir.split('/').pop();
			if (!this.argv0) {
				this.argv0 = basename + '.wasm';
			}
			resp = await fetch(data.Bin);
		} else {
			if (!this.argv0) {
				const match = /\/([^\/]+$)/.exec(this.href);
				this.argv0 = match ? match[1] : this.href;
			}
			setTitle(this.argv0);
		}

		if (parseContentType(resp) === 'text/plain') { // TODO support text/html formatted error
			const el = this.el || document.body;
			el.innerHTML = `<pre class="buildLog"></pre>`;
			el.querySelector('pre').innerText = await resp.text();
			return;
		}

		this.module = await WebAssembly.compileStreaming(resp);

		if (this.el && !this.args) {
			return this.interact();
		}

		let argv = [this.argv0];
		if (this.args) {
			argv = argv.concat(this.args);
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
		go.env = this.env;
		go.argv = argv;
		const instance = await WebAssembly.instantiate(this.module, go.importObject);
		await go.run(instance);
	}
};

global.goRun = (() => {
	const cfg = GoRunner.parseConfigData(document.currentScript);
	return new GoRunner(cfg);
})();
