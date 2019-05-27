# `gorunwasm` -- An HTTP Server Wrapper For Go

## What

The server:

- Provides a `main.wasm` endpoint that automatically builds and caches a built
  WASM binary from an input Go package; cache is invalidated when any of the
  source go files are changed, causing a re-build on next load.
- Provides a `build.log` endpoint exposing the build log; especially useful
  when the build fails.
- Also provides a `build.json` endpoint to see target
  [`build.Package`][golang_build_package] data.

The frontend:

- Provides a simple harness that instantiates the built wasm binary ...
- ... or instead shows you the build log on failure.
- Provides a simple `argv` input box, allowing the target command to be ran as
  many times as makes sense.

The Demo:

- `gorunwasm` itself has a build-flagged `main()` wasm entry point in
  [`wasm_main.go`](wasm_main.go); it is a simple DOM manipulating example.

[golang_build_package]: https://golang.org/pkg/go/build/#Package
