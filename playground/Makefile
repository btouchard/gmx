.PHONY: wasm serve clean

DOCS_PLAYGROUND = ../docs/playground

wasm:
	@echo "Building WASM binary..."
	cd .. && GOOS=js GOARCH=wasm go build -o docs/playground/gmx.wasm ./playground/
	@echo "Copying wasm_exec.js..."
	@if [ -f "$$(go env GOROOT)/lib/wasm/wasm_exec.js" ]; then \
		cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(DOCS_PLAYGROUND)/; \
	elif [ -f "$$(go env GOROOT)/misc/wasm/wasm_exec.js" ]; then \
		cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" $(DOCS_PLAYGROUND)/; \
	else \
		echo "Error: wasm_exec.js not found"; exit 1; \
	fi
	@echo "Build complete: $(DOCS_PLAYGROUND)/gmx.wasm"

serve: wasm
	@echo "Playground running at http://localhost:8080"
	@cd $(DOCS_PLAYGROUND) && python3 -m http.server 8080

clean:
	@echo "Cleaning up..."
	rm -f $(DOCS_PLAYGROUND)/gmx.wasm $(DOCS_PLAYGROUND)/wasm_exec.js
	@echo "Clean complete"
