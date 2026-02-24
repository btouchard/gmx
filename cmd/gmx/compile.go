package main

import (
	"fmt"
	"github.com/btouchard/gmx/internal/compiler/generator"
	"github.com/btouchard/gmx/internal/compiler/lexer"
	"github.com/btouchard/gmx/internal/compiler/parser"
	"github.com/btouchard/gmx/internal/compiler/resolver"
	"os"
	"path/filepath"
	"strings"
)

// compile reads a .gmx file and returns the generated Go source code.
func compile(inputFile string) (string, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	// 1. Lexing
	l := lexer.New(string(data))

	// 2. Parsing
	p := parser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		var b strings.Builder
		b.WriteString("parser errors:\n")
		for _, e := range p.Errors() {
			b.WriteString("  " + e + "\n")
		}
		return "", fmt.Errorf("%s", b.String())
	}

	// 3. Import Resolution & Generation
	gen := generator.New()

	if len(file.Imports) > 0 {
		basePath := filepath.Dir(inputFile)
		absInputFile, err := filepath.Abs(inputFile)
		if err != nil {
			return "", fmt.Errorf("resolving input file path: %w", err)
		}

		res := resolver.New(basePath)
		resolved, resolveErrors := res.Resolve(file, absInputFile)

		if len(resolveErrors) > 0 {
			var b strings.Builder
			b.WriteString("import resolution errors:\n")
			for _, e := range resolveErrors {
				b.WriteString("  " + e + "\n")
			}
			return "", fmt.Errorf("%s", b.String())
		}

		code, err := gen.GenerateResolved(resolved)
		if err != nil {
			return "", fmt.Errorf("generation: %w", err)
		}
		return code, nil
	}

	code, err := gen.Generate(file)
	if err != nil {
		return "", fmt.Errorf("generation: %w", err)
	}
	return code, nil
}
