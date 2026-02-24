package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: gmx run <input.gmx> [-- args...]\n")
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	inputFile := fs.Arg(0)

	// Collect arguments after "--" to pass to the binary
	var extraArgs []string
	allArgs := fs.Args()
	for i, a := range allArgs[1:] {
		if a == "--" {
			extraArgs = allArgs[i+2:]
			break
		}
	}

	// Build to a temporary binary
	tmpDir, err := os.MkdirTemp("", "gmx-run-*")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	base := filepath.Base(inputFile)
	binaryName := strings.TrimSuffix(base, filepath.Ext(base))
	binaryPath := filepath.Join(tmpDir, binaryName)

	// Cleanup on exit
	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}
	defer cleanup()

	if err := buildBinary(inputFile, binaryPath); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run the binary, forwarding stdin/stdout/stderr
	cmd := exec.Command(binaryPath, extraArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward signals to the child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error starting binary: %v\n", err)
		os.Exit(1)
	}

	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			_ = cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
