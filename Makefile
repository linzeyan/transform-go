all: wasm build
.PHONY: all

wasm:
	GOOS=js GOARCH=wasm go build -o web/app.wasm ./wasm
.PHONY: wasm

build:
	go build .
.PHONY: build