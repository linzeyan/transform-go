PKG := ./pkg/convert
FUZZ_FUNCS := $(shell grep -hoE '^func[[:space:]]+(Fuzz[[:alnum:]_]+)' $(PKG)/*_test.go | awk '{print $$2}')

all: test wasm build
.PHONY: all

wasm:
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o web/app.wasm ./wasm
# 	wasm-opt web/app.wasm -Oz --enable-bulk-memory -o web/app.wasm
# 	wasm-opt web/app.wasm --enable-bulk-memory --metrics

.PHONY: wasm

build:
	go build .
.PHONY: build

test:
	go test -cover ./...
.PHONY: test

benchmark:
	go test -bench=. -benchmem -count=2 ./...
.PHONY: benchmark

fuzz:
	@for f in $(FUZZ_FUNCS); do \
		echo "==> Running $$f"; \
		go test -fuzz=$$f -run=^$$f -fuzztime=30s $(PKG) || exit $$?; \
	done
.PHONY: fuzz
