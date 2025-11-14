FUZZTIME ?= 30s

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
	@go list -f '{{.Dir}} {{.ImportPath}}' ./pkg/... | while read -r dir pkg; do \
		if ls $$dir/*_test.go >/dev/null 2>&1 && grep -qE '^func[[:space:]]+Fuzz' $$dir/*_test.go; then \
			echo "==> Running fuzzers in $$pkg"; \
			for f in $$(grep -hoE '^func[[:space:]]+(Fuzz[[:alnum:]_]+)' $$dir/*_test.go | awk '{print $$2}'); do \
				echo "  -> $$f"; \
				go test -run=^$$f -fuzz=$$f -fuzztime=$(FUZZTIME) $$pkg || exit $$?; \
			done; \
		fi; \
	done
.PHONY: fuzz
