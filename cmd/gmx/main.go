package main

import (
	"flag"
	"fmt"
	"gmx/internal/compiler/generator"
	"gmx/internal/compiler/lexer"
	"gmx/internal/compiler/parser"
	"gmx/internal/compiler/resolver"
	"os"
	"path/filepath"
)

func main() {
	var outputFile string
	flag.StringVar(&outputFile, "o", "main.go", "output file path")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gmx [-o output.go] <input.gmx>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	input := string(data)

	// 1. Lexing
	l := lexer.New(input)

	// 2. Parsing
	p := parser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser Errors:")
		for _, err := range p.Errors() {
			fmt.Println(err)
		}
		os.Exit(1)
	}

	// 3. Import Resolution (if imports exist)
	var code string
	gen := generator.New()

	if len(file.Imports) > 0 {
		// Multi-file compilation with imports
		basePath := filepath.Dir(inputFile)

		// Convert inputFile to absolute path for resolver
		absInputFile, err := filepath.Abs(inputFile)
		if err != nil {
			fmt.Printf("Error resolving input file path: %v\n", err)
			os.Exit(1)
		}

		res := resolver.New(basePath)
		resolved, resolveErrors := res.Resolve(file, absInputFile)

		if len(resolveErrors) > 0 {
			fmt.Println("Import Resolution Errors:")
			for _, err := range resolveErrors {
				fmt.Println(err)
			}
			os.Exit(1)
		}

		// 4. Generation with resolved imports
		code, err = gen.GenerateResolved(resolved)
		if err != nil {
			fmt.Printf("Generation Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Single-file compilation (backward compatibility)
		code, err = gen.Generate(file)
		if err != nil {
			fmt.Printf("Generation Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Create parent directories if needed
	if dir := filepath.Dir(outputFile); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating output directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Write to file
	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s successfully\n", outputFile)
}
