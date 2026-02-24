package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "build":
		cmdBuild(args)
	case "run":
		cmdRun(args)
	case "fmt":
		cmdFmt(args)
	default:
		// Fallback: if an argument looks like a .gmx file, treat as "build"
		if strings.HasSuffix(cmd, ".gmx") {
			cmdBuild(os.Args[1:])
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	bin := filepath.Base(os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, `Usage: %s <command> [arguments]

Commands:
  build   Compile a .gmx file into a Go binary
  run     Build and run a .gmx file immediately
  fmt     Format .gmx files

Run '%s <command> -h' for command-specific help.

Shortcut:
  %s <file.gmx>   Equivalent to '%s build <file.gmx>'
`, bin, bin, bin, bin)
}
