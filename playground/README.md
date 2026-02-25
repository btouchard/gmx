# GMX Playground

Browser-based interactive editor that compiles `.gmx` files to Go in real-time using WebAssembly.

## Features

- Live compilation from `.gmx` to Go
- Syntax highlighting for GMX and Go
- Dark theme inspired by VS Code
- Resizable split-pane editor
- Auto-compile on keystroke (with debounce)
- Multiple example presets
- No build tools, no npm, pure vanilla HTML/JS

## Quick Start

```bash
# Build the WASM binary
make wasm

# Serve the playground (http://localhost:8080)
make serve
```

Then open http://localhost:8080 in your browser.

## Requirements

- Go 1.24+
- Python 3 (for the local server)
- Modern browser with WebAssembly support (Chrome, Firefox, Safari, Edge)

## Build Targets

```bash
make wasm    # Build WASM binary and copy wasm_exec.js
make serve   # Build and serve on http://localhost:8080
make clean   # Remove generated files
```

## Architecture

### WASM Bridge (`main_wasm.go`)

Compiles to WebAssembly and exposes a single function to JavaScript:

```js
compileGMX(source) -> { code: string, errors: []string }
```

The bridge uses the GMX compiler pipeline:
1. Lexer: tokenize source
2. Parser: build AST
3. Generator: emit Go code

Import resolution is disabled (no multi-file support in the playground).

### Web UI (`web/`)

- **index.html** — Single-page app layout
- **playground.js** — WASM initialization, CodeMirror setup, compilation logic
- **style.css** — Dark theme styling
- **gmx-lang.js** — CodeMirror 6 language mode for `.gmx` syntax
- **wasm_exec.js** — Go WASM runtime (copied from GOROOT)

All CodeMirror dependencies are loaded from CDN (esm.sh) via ES modules.

## Development

The playground runs entirely in the browser. After building the WASM binary, you can open `index.html` directly in a browser (if served with correct MIME types).

Python's built-in HTTP server is used for local development:

```bash
cd web
python3 -m http.server 8080
```

## Limitations

- No multi-file imports (resolver is skipped)
- No service execution (compile-time only)
- No database operations (WASM binary is pure compiler)

## Browser Compatibility

Tested on:
- Chrome/Edge 90+
- Firefox 89+
- Safari 15+

Requires WebAssembly and ES6 module support.
