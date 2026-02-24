package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Top-level section tags must be at column 0 (start of line)
var (
	openTagRe  = regexp.MustCompile(`^<(script|template|style)(\s+scoped)?>$`)
	closeTagRe = regexp.MustCompile(`^</(script|template|style)>$`)
)

func cmdFmt(args []string) {
	fs := flag.NewFlagSet("fmt", flag.ExitOnError)
	diff := fs.Bool("d", false, "display diff instead of writing")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: gmx fmt [-d] <files...>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	exitCode := 0
	for _, file := range fs.Args() {
		if err := fmtFile(file, *diff); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", file, err)
			exitCode = 1
		}
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

type section struct {
	tag     string
	attr    string // e.g. " scoped"
	content string
}

// parseSections extracts top-level sections from a .gmx file.
// Only tags at column 0 (start of line) are considered section boundaries.
func parseSections(input string) []section {
	lines := strings.Split(input, "\n")
	var sections []section
	var current *section
	var contentLines []string

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")

		if m := openTagRe.FindStringSubmatch(trimmed); m != nil && current == nil {
			current = &section{tag: m[1], attr: m[2]}
			contentLines = nil
			continue
		}

		if m := closeTagRe.FindStringSubmatch(trimmed); m != nil && current != nil && m[1] == current.tag {
			current.content = strings.Join(contentLines, "\n")
			sections = append(sections, *current)
			current = nil
			contentLines = nil
			continue
		}

		if current != nil {
			contentLines = append(contentLines, line)
		}
	}

	return sections
}

func fmtFile(path string, showDiff bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	original := string(data)
	sections := parseSections(original)

	if len(sections) == 0 {
		return fmt.Errorf("no sections found")
	}

	// Order: script, template, style (preserving multiples of same type)
	order := []string{"script", "template", "style"}
	var ordered []section
	for _, tag := range order {
		for _, s := range sections {
			if s.tag == tag {
				ordered = append(ordered, s)
			}
		}
	}

	// Rebuild file
	var b strings.Builder
	for i, s := range ordered {
		if i > 0 {
			b.WriteString("\n")
		}
		formatted := formatSection(s)
		b.WriteString(formatted)
		b.WriteString("\n")
	}

	result := b.String()

	if showDiff {
		if result != original {
			fmt.Printf("--- %s\n+++ %s (formatted)\n", path, path)
			printSimpleDiff(original, result)
		}
		return nil
	}

	if result == original {
		return nil
	}

	return os.WriteFile(path, []byte(result), 0644)
}

func formatSection(s section) string {
	var b strings.Builder
	b.WriteString("<" + s.tag + s.attr + ">\n")

	lines := strings.Split(s.content, "\n")

	// Trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	// Trim leading empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}

	// Find minimum indentation (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	// Re-indent with 2 spaces based on relative indentation
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			b.WriteString("\n")
			continue
		}
		trimmed := line
		if minIndent > 0 && len(line) >= minIndent {
			trimmed = line[minIndent:]
		}
		// Apply 2-space base indent
		b.WriteString("  " + strings.TrimRight(trimmed, " \t") + "\n")
	}

	b.WriteString("</" + s.tag + ">")
	return b.String()
}

func printSimpleDiff(a, b string) {
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")

	maxLen := max(len(aLines), len(bLines))

	for i := range maxLen {
		aLine, bLine := "", ""
		if i < len(aLines) {
			aLine = aLines[i]
		}
		if i < len(bLines) {
			bLine = bLines[i]
		}
		if aLine != bLine {
			if i < len(aLines) {
				fmt.Printf("-%s\n", aLine)
			}
			if i < len(bLines) {
				fmt.Printf("+%s\n", bLine)
			}
		}
	}
}
