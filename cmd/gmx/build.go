package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	outputBinary := fs.String("o", "", "output binary path (default: input filename without extension)")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: gmx build [-o binary] <input.gmx>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	binary := *outputBinary
	if binary == "" {
		base := filepath.Base(inputFile)
		binary = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if err := buildBinary(inputFile, binary); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Built %s successfully\n", binary)
}

// buildBinary compiles a .gmx file into a Go binary.
func buildBinary(inputFile, outputBinary string) error {
	code, err := compile(inputFile)
	if err != nil {
		return err
	}

	// Create a temporary directory for the build
	tmpDir, err := os.MkdirTemp("", "gmx-build-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	// Write generated Go source
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("writing generated code: %w", err)
	}

	// Initialize go.mod in the temp directory
	modInit := exec.Command("go", "mod", "init", "gmx-app")
	modInit.Dir = tmpDir
	modInit.Stderr = os.Stderr
	if err := modInit.Run(); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// Run go mod tidy to resolve dependencies
	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = tmpDir
	modTidy.Stderr = os.Stderr
	if err := modTidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// Build the binary
	absBinary, err := filepath.Abs(outputBinary)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	// Create an output directory if needed
	if dir := filepath.Dir(absBinary); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	goBuild := exec.Command("go", "build", "-o", absBinary, ".")
	goBuild.Dir = tmpDir
	goBuild.Stdout = os.Stdout
	goBuild.Stderr = os.Stderr
	if err := goBuild.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	return nil
}
