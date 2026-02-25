//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/btouchard/gmx/internal/compiler/generator"
	"github.com/btouchard/gmx/internal/compiler/lexer"
	"github.com/btouchard/gmx/internal/compiler/parser"
)

func main() {
	js.Global().Set("compileGMX", js.FuncOf(compileGMXWrapper))

	// Keep the program alive
	select {}
}

// compileGMXWrapper wraps the compilation logic with panic recovery
func compileGMXWrapper(this js.Value, args []js.Value) interface{} {
	var result map[string]interface{}

	defer func() {
		if r := recover(); r != nil {
			// Set error result on panic
			result = make(map[string]interface{})
			result["code"] = ""
			result["errors"] = []interface{}{fmt.Sprintf("panic: %v", r)}
		}
	}()

	if len(args) != 1 {
		result = make(map[string]interface{})
		result["code"] = ""
		result["errors"] = []interface{}{"expected 1 argument (source code)"}
		return js.ValueOf(result)
	}

	source := args[0].String()
	code, errors := compileGMX(source)

	result = make(map[string]interface{})
	result["code"] = code

	// Convert error strings to JS array
	jsErrors := make([]interface{}, len(errors))
	for i, err := range errors {
		jsErrors[i] = err
	}
	result["errors"] = jsErrors

	return js.ValueOf(result)
}

// compileGMX compiles a .gmx source string and returns the generated Go code and any errors
func compileGMX(source string) (string, []string) {
	var errors []string

	// 1. Lexing
	l := lexer.New(source)

	// 2. Parsing
	p := parser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		return "", p.Errors()
	}

	// 3. Generation (skip resolver - no multi-file support in playground)
	gen := generator.New()

	// If there are imports, warn the user
	if len(file.Imports) > 0 {
		var importNames []string
		for _, imp := range file.Imports {
			importNames = append(importNames, imp.Path)
		}
		errors = append(errors,
			fmt.Sprintf("warning: imports are not supported in playground (%s)",
				strings.Join(importNames, ", ")))
	}

	code, err := gen.Generate(file)
	if err != nil {
		errors = append(errors, fmt.Sprintf("generation error: %v", err))
		return "", errors
	}

	return code, errors
}
