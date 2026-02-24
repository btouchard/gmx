package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/btouchard/gmx/internal/compiler/ast"
	"github.com/btouchard/gmx/internal/compiler/lexer"
	"github.com/btouchard/gmx/internal/compiler/parser"
)

// ComponentInfo stores metadata about an imported component
type ComponentInfo struct {
	File *ast.GMXFile // parsed component file
	Path string       // absolute path (for debugging)
	Name string       // import name (e.g., "TaskItem")
}

// ResolvedFile represents a GMX file with all imports resolved
type ResolvedFile struct {
	Main       *ast.GMXFile              // enriched main file (merged declarations)
	Components map[string]*ComponentInfo // component metadata for templates
}

// Resolver handles recursive import resolution for .gmx files
type Resolver struct {
	basePath string                  // directory of root .gmx file
	parsed   map[string]*ast.GMXFile // cache: absolute path → parsed AST
	loading  map[string]bool         // circular import detection
	errors   []string
}

// New creates a new Resolver with the specified base path for resolving relative imports
func New(basePath string) *Resolver {
	return &Resolver{
		basePath: basePath,
		parsed:   make(map[string]*ast.GMXFile),
		loading:  make(map[string]bool),
		errors:   []string{},
	}
}

// Errors returns all accumulated errors during resolution
func (r *Resolver) Errors() []string {
	return r.errors
}

// addError accumulates an error message
func (r *Resolver) addError(format string, args ...interface{}) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

// resolvePath converts a GMX import path to an absolute file system path
func (r *Resolver) resolvePath(importPath string, currentDir string) (string, error) {
	// Skip native Go imports
	if !strings.HasSuffix(importPath, ".gmx") {
		return "", fmt.Errorf("not a .gmx file: %s", importPath)
	}

	// Resolve relative to currentDir
	absPath := filepath.Join(currentDir, importPath)
	absPath = filepath.Clean(absPath)

	// Convert to absolute path for consistent caching
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %s: %w", importPath, err)
	}

	return absPath, nil
}

// loadFile reads and parses a .gmx file (with caching)
func (r *Resolver) loadFile(absPath string) (*ast.GMXFile, error) {
	// Check cache first
	if cached, ok := r.parsed[absPath]; ok {
		return cached, nil
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}

	// Parse file
	l := lexer.New(string(data))
	p := parser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s: %v", absPath, p.Errors())
	}

	// Cache the file
	r.parsed[absPath] = file

	return file, nil
}

// Resolve resolves all imports in a GMX file recursively
func (r *Resolver) Resolve(main *ast.GMXFile, mainPath string) (*ResolvedFile, []string) {
	resolved := &ResolvedFile{
		Main:       &ast.GMXFile{}, // create new file to accumulate merged declarations
		Components: make(map[string]*ComponentInfo),
	}

	// Copy main file's non-import declarations
	resolved.Main.Models = append([]*ast.ModelDecl{}, main.Models...)
	resolved.Main.Services = append([]*ast.ServiceDecl{}, main.Services...)
	resolved.Main.Vars = append([]*ast.VarDecl{}, main.Vars...)
	resolved.Main.Template = main.Template
	resolved.Main.Style = main.Style

	// Copy script block (functions will be merged later)
	if main.Script != nil {
		resolved.Main.Script = &ast.ScriptBlock{
			Source:    main.Script.Source,
			Funcs:     append([]*ast.FuncDecl{}, main.Script.Funcs...),
			StartLine: main.Script.StartLine,
		}
	}

	// Get directory of main file for relative imports
	mainDir := filepath.Dir(mainPath)

	// Process each import
	for _, imp := range main.Imports {
		if imp.IsNative {
			// Native Go imports pass through unchanged
			resolved.Main.Imports = append(resolved.Main.Imports, imp)
			continue
		}

		// Resolve .gmx import
		if err := r.resolveImport(imp, mainDir, resolved); err != nil {
			r.addError("failed to resolve import %s: %v", imp.Path, err)
		}
	}

	return resolved, r.errors
}

// resolveImport handles a single .gmx import (recursive)
func (r *Resolver) resolveImport(imp *ast.ImportDecl, currentDir string, resolved *ResolvedFile) error {
	// Resolve path
	absPath, err := r.resolvePath(imp.Path, currentDir)
	if err != nil {
		return err
	}

	// Check for circular imports BEFORE loading
	if r.loading[absPath] {
		return fmt.Errorf("circular import detected: %s", absPath)
	}

	// Mark as loading (circular import detection)
	r.loading[absPath] = true
	defer delete(r.loading, absPath) // cleanup when done with this import chain

	// Load and parse file
	file, err := r.loadFile(absPath)
	if err != nil {
		return err
	}

	// Recursively resolve this file's imports
	importedDir := filepath.Dir(absPath)
	for _, nestedImp := range file.Imports {
		if nestedImp.IsNative {
			// Add to main's imports if not duplicate
			if !r.hasImport(resolved.Main, nestedImp) {
				resolved.Main.Imports = append(resolved.Main.Imports, nestedImp)
			}
			continue
		}

		// Recursively resolve nested .gmx import
		if err := r.resolveImport(nestedImp, importedDir, resolved); err != nil {
			return fmt.Errorf("nested import from %s: %w", absPath, err)
		}
	}

	// Handle based on import type
	if imp.Default != "" {
		// Default import: component
		return r.resolveDefaultImport(imp, file, absPath, resolved)
	} else if len(imp.Members) > 0 {
		// Destructured import: specific exports
		return r.resolveDestructuredImport(imp, file, resolved)
	}

	return nil
}

// hasImport checks if an import already exists in the file
func (r *Resolver) hasImport(file *ast.GMXFile, imp *ast.ImportDecl) bool {
	for _, existing := range file.Imports {
		if existing.Path == imp.Path && existing.Alias == imp.Alias {
			return true
		}
	}
	return false
}

// resolveDefaultImport handles: import TaskItem from './components/TaskItem.gmx'
func (r *Resolver) resolveDefaultImport(imp *ast.ImportDecl, file *ast.GMXFile, absPath string, resolved *ResolvedFile) error {
	// Validate: default imports must have templates (they're components)
	if file.Template == nil {
		return fmt.Errorf("default import %s has no template (not a valid component)", imp.Default)
	}

	// Store component for template composition
	resolved.Components[imp.Default] = &ComponentInfo{
		File: file,
		Path: absPath,
		Name: imp.Default,
	}

	// Merge models (not functions - components are self-contained)
	for _, model := range file.Models {
		if !r.hasModel(resolved.Main, model.Name) {
			resolved.Main.Models = append(resolved.Main.Models, model)
		} else {
			r.addError("warning: model %s already defined, skipping import from %s", model.Name, absPath)
		}
	}

	// Merge services
	for _, service := range file.Services {
		if !r.hasService(resolved.Main, service.Name) {
			resolved.Main.Services = append(resolved.Main.Services, service)
		} else {
			r.addError("warning: service %s already defined, skipping import from %s", service.Name, absPath)
		}
	}

	return nil
}

// resolveDestructuredImport handles: import { sendEmail, MailerConfig } from './services/mailer.gmx'
func (r *Resolver) resolveDestructuredImport(imp *ast.ImportDecl, file *ast.GMXFile, resolved *ResolvedFile) error {
	for _, memberName := range imp.Members {
		found := false

		// Try to find in models
		for _, model := range file.Models {
			if model.Name == memberName {
				if !r.hasModel(resolved.Main, model.Name) {
					resolved.Main.Models = append(resolved.Main.Models, model)
					found = true
				} else {
					r.addError("warning: model %s already defined, skipping", model.Name)
					found = true
				}
				break
			}
		}
		if found {
			continue
		}

		// Try to find in services
		for _, service := range file.Services {
			if service.Name == memberName {
				if !r.hasService(resolved.Main, service.Name) {
					resolved.Main.Services = append(resolved.Main.Services, service)
					found = true
				} else {
					r.addError("warning: service %s already defined, skipping", service.Name)
					found = true
				}
				break
			}
		}
		if found {
			continue
		}

		// Try to find in functions
		if file.Script != nil {
			for _, fn := range file.Script.Funcs {
				if fn.Name == memberName {
					if resolved.Main.Script == nil {
						resolved.Main.Script = &ast.ScriptBlock{}
					}
					resolved.Main.Script.Funcs = append(resolved.Main.Script.Funcs, fn)
					found = true
					break
				}
			}
		}

		if !found {
			return fmt.Errorf("imported member %s not found in %s", memberName, imp.Path)
		}
	}

	return nil
}

// Helper functions for duplicate detection
func (r *Resolver) hasModel(file *ast.GMXFile, name string) bool {
	for _, m := range file.Models {
		if m.Name == name {
			return true
		}
	}
	return false
}

func (r *Resolver) hasService(file *ast.GMXFile, name string) bool {
	for _, s := range file.Services {
		if s.Name == name {
			return true
		}
	}
	return false
}
