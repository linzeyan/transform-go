# transform-go

A lightweight Go-powered clone of [transform](https://github.com/ritz078/transform) focused on converting JSON, YAML, TOML, Go structs, and JSON Schema right inside the browser via WebAssembly.

## Features
- Client-side conversions powered by a Go → WebAssembly module
- Round-trip transformations between JSON, Go structs, YAML, TOML, and JSON Schema
- Modern UI inspired by transform.tools with keyboard shortcuts and copy helpers

## Development
```bash
# build the wasm module
GOOS=js GOARCH=wasm go build -o web/app.wasm ./wasm

# run the dev server
go run .
```
Visit [http://localhost:8880](http://localhost:8880) to try the UI.

## Inspiration
This project is heavily inspired by the amazing work in [ritz078/transform](https://github.com/ritz078/transform).
