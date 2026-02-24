package generator

import (
	"fmt"
	"github.com/btouchard/gmx/internal/compiler/ast"
	"github.com/btouchard/gmx/internal/compiler/resolver"
	"github.com/btouchard/gmx/internal/compiler/script"
	"go/format"
	"strings"
)

type Generator struct {
	// Will hold state for more complex generation in later phases
}

func New() *Generator {
	return &Generator{}
}

// GenerateResolved generates Go code from a resolved GMX file with imports
func (g *Generator) GenerateResolved(resolved *resolver.ResolvedFile) (string, error) {
	// Use the internal method on the merged Main file with components
	return g.generateWithComponents(resolved.Main, resolved.Components)
}

// Generate takes a GMXFile AST and produces complete, compilable Go source code
// This method is kept for backward compatibility (single-file compilation)
func (g *Generator) Generate(file *ast.GMXFile) (string, error) {
	// No components for single-file compilation
	return g.generateWithComponents(file, nil)
}

// generateWithComponents is the internal implementation that handles both single-file and multi-file compilation
func (g *Generator) generateWithComponents(file *ast.GMXFile, components map[string]*resolver.ComponentInfo) (string, error) {
	var b strings.Builder

	// Compute routes ONCE at the beginning
	var routes map[string]string
	if file.Template != nil {
		routes = g.genRouteRegistry(file.Template.Source)
	} else {
		routes = make(map[string]string)
	}

	// Package declaration
	b.WriteString("package main\n\n")

	// Imports
	b.WriteString(g.genImports(file))
	b.WriteString("\n")

	// Helper functions (if needed)
	helpers := g.genHelpers(file)
	if helpers != "" {
		b.WriteString(helpers)
	}

	// Variables (if any)
	if len(file.Vars) > 0 {
		b.WriteString("// ========== Variables ==========\n\n")
		b.WriteString(g.genVars(file.Vars))
		b.WriteString("\n")
	}

	// Models (if any)
	if len(file.Models) > 0 {
		b.WriteString("// ========== Models ==========\n\n")
		b.WriteString(g.genModels(file.Models))
		b.WriteString("\n")
	}

	// Services (if any)
	if len(file.Services) > 0 {
		b.WriteString("// ========== Services ==========\n\n")
		b.WriteString(g.genServices(file.Services))
		b.WriteString("\n")
	}

	// Script (transpiled functions)
	if file.Script != nil && file.Script.Funcs != nil {
		b.WriteString("// ========== Script (Transpiled) ==========\n\n")
		modelNames := g.extractModelNames(file.Models)
		result := script.Transpile(file.Script, modelNames)
		if len(result.Errors) > 0 {
			return "", fmt.Errorf("transpile errors: %v", result.Errors)
		}
		b.WriteString(result.GoCode)
		b.WriteString("\n")

		// Generate HTTP handler wrappers
		b.WriteString("// ========== Script Handler Wrappers ==========\n\n")
		b.WriteString(g.genScriptHandlers(file.Script))
		b.WriteString("\n")
	}

	// Template setup
	if file.Template != nil {
		b.WriteString("// ========== Template ==========\n\n")
		b.WriteString(g.genTemplateInit(routes))
		b.WriteString("\n")
		b.WriteString(g.genTemplateConst(file, components))
		b.WriteString("\n")
	}

	// Page Data struct
	if len(file.Models) > 0 {
		b.WriteString("// ========== Page Data ==========\n\n")
		b.WriteString(g.genPageData(file.Models))
		b.WriteString("\n")
	}

	// Database variable (declare at package level if models exist)
	if len(file.Models) > 0 {
		b.WriteString("// ========== Database ==========\n\n")
		b.WriteString("var db *gorm.DB\n\n")
	}

	// Handlers
	if file.Template != nil {
		b.WriteString("// ========== Handlers ==========\n\n")
		b.WriteString(g.genHandlers(file, routes))
		b.WriteString("\n")
	}

	// Main function
	b.WriteString("// ========== Main ==========\n\n")
	b.WriteString(g.genMain(file, routes))

	// Format the generated code
	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		// If formatting fails, return the unformatted code with error for debugging
		return b.String(), fmt.Errorf("format error: %w", err)
	}

	return string(formatted), nil
}
