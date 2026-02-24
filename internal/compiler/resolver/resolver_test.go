package resolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/btouchard/gmx/internal/compiler/ast"
	"github.com/btouchard/gmx/internal/compiler/lexer"
	"github.com/btouchard/gmx/internal/compiler/parser"
)

// Helper to parse a .gmx file for testing
func parseFile(t *testing.T, path string) *ast.GMXFile {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors in %s: %v", path, p.Errors())
	}

	return file
}

func TestSimpleComponentImport(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create component file
	componentPath := filepath.Join(tmpDir, "TaskItem.gmx")
	componentContent := `<script>
model Task {
  id: uuid @pk
  title: string
}
</script>

<template>
<div>{{.Title}}</div>
</template>

<style scoped>
div { color: blue; }
</style>`
	os.WriteFile(componentPath, []byte(componentContent), 0644)

	// Create main file
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import TaskItem from "./TaskItem.gmx"
</script>

<template>
<div>{{template "TaskItem" .}}</div>
</template>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	// Parse main
	file := parseFile(t, mainPath)

	// Resolve
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, mainPath)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify component registered
	if _, ok := resolved.Components["TaskItem"]; !ok {
		t.Error("TaskItem component not found")
	}

	// Verify model merged
	found := false
	for _, m := range resolved.Main.Models {
		if m.Name == "Task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Task model not merged")
	}
}

func TestDestructuredImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mailer service
	mailerPath := filepath.Join(tmpDir, "mailer.gmx")
	mailerContent := `<script>
service Mailer {
  provider: "smtp"
  host: string @env("SMTP_HOST")
}

func sendEmail(to: string) error {
  return error("not implemented")
}
</script>`
	os.WriteFile(mailerPath, []byte(mailerContent), 0644)

	// Create main
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import { sendEmail } from "./mailer.gmx"

func notifyUser() error {
  return try sendEmail("user@example.com")
}
</script>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, mainPath)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify sendEmail function merged
	found := false
	if resolved.Main.Script != nil {
		for _, fn := range resolved.Main.Script.Funcs {
			if fn.Name == "sendEmail" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("sendEmail function not merged")
	}
}

func TestDestructuredImportMultipleMembers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create utils file with both model and service
	utilsPath := filepath.Join(tmpDir, "utils.gmx")
	utilsContent := `<script>
model Config {
  id: uuid @pk
  key: string
}

service Logger {
  provider: "console"
  level: string @env("LOG_LEVEL")
}

func logMessage(msg: string) error {
  return error("not implemented")
}
</script>`
	os.WriteFile(utilsPath, []byte(utilsContent), 0644)

	// Create main importing multiple members
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import { Config, Logger, logMessage } from "./utils.gmx"
</script>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, mainPath)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify all members merged
	hasConfig := false
	for _, m := range resolved.Main.Models {
		if m.Name == "Config" {
			hasConfig = true
			break
		}
	}
	if !hasConfig {
		t.Error("Config model not merged")
	}

	hasLogger := false
	for _, s := range resolved.Main.Services {
		if s.Name == "Logger" {
			hasLogger = true
			break
		}
	}
	if !hasLogger {
		t.Error("Logger service not merged")
	}

	hasLogMessage := false
	if resolved.Main.Script != nil {
		for _, fn := range resolved.Main.Script.Funcs {
			if fn.Name == "logMessage" {
				hasLogMessage = true
				break
			}
		}
	}
	if !hasLogMessage {
		t.Error("logMessage function not merged")
	}
}

func TestTransitiveImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create C file with a model
	cPath := filepath.Join(tmpDir, "c.gmx")
	cContent := `<script>
model BaseModel {
  id: uuid @pk
}
</script>

<template>
<div>C</div>
</template>`
	os.WriteFile(cPath, []byte(cContent), 0644)

	// Create B file that imports C
	bPath := filepath.Join(tmpDir, "b.gmx")
	bContent := `<script>
import C from "./c.gmx"
</script>

<template>
<div>B: {{template "C" .}}</div>
</template>`
	os.WriteFile(bPath, []byte(bContent), 0644)

	// Create A file that imports B
	aPath := filepath.Join(tmpDir, "a.gmx")
	aContent := `<script>
import B from "./b.gmx"
</script>

<template>
<div>A: {{template "B" .}}</div>
</template>`
	os.WriteFile(aPath, []byte(aContent), 0644)

	file := parseFile(t, aPath)
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, aPath)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify all components are registered
	if _, ok := resolved.Components["B"]; !ok {
		t.Error("B component not found")
	}
	if _, ok := resolved.Components["C"]; !ok {
		t.Error("C component not found")
	}

	// Verify BaseModel from C is merged
	found := false
	for _, m := range resolved.Main.Models {
		if m.Name == "BaseModel" {
			found = true
			break
		}
	}
	if !found {
		t.Error("BaseModel from C not merged transitively")
	}
}

func TestCircularImportDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// A imports B
	aPath := filepath.Join(tmpDir, "a.gmx")
	aContent := `<script>
import B from "./b.gmx"
</script>
<template><div>A</div></template>`
	os.WriteFile(aPath, []byte(aContent), 0644)

	// B imports A (circular!)
	bPath := filepath.Join(tmpDir, "b.gmx")
	bContent := `<script>
import A from "./a.gmx"
</script>
<template><div>B</div></template>`
	os.WriteFile(bPath, []byte(bContent), 0644)

	file := parseFile(t, aPath)
	res := New(tmpDir)
	_, errors := res.Resolve(file, aPath)

	// Expect circular import error
	if len(errors) == 0 {
		t.Error("expected circular import error")
	}

	foundCircular := false
	for _, err := range errors {
		if strings.Contains(err, "circular import") {
			foundCircular = true
			break
		}
	}
	if !foundCircular {
		t.Errorf("expected 'circular import' error, got: %v", errors)
	}
}

func TestDuplicateModel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create component with Task model
	componentPath := filepath.Join(tmpDir, "component.gmx")
	componentContent := `<script>
model Task {
  id: uuid @pk
  title: string
}
</script>

<template>
<div>Component</div>
</template>`
	os.WriteFile(componentPath, []byte(componentContent), 0644)

	// Create main with Task model (duplicate)
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import Component from "./component.gmx"

model Task {
  id: uuid @pk
  name: string
}
</script>

<template>
<div>Main</div>
</template>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, mainPath)

	// Should have warning about duplicate
	if len(errors) == 0 {
		t.Error("expected warning about duplicate model")
	}

	// First definition (from main) should win
	found := false
	for _, m := range resolved.Main.Models {
		if m.Name == "Task" {
			found = true
			// Check it's from main (has "name" field, not "title")
			hasName := false
			for _, field := range m.Fields {
				if field.Name == "name" {
					hasName = true
					break
				}
			}
			if !hasName {
				t.Error("Expected main's Task model (with 'name' field) to be kept")
			}
			break
		}
	}
	if !found {
		t.Error("Task model not found")
	}
}

func TestMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main that imports non-existent file
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import Missing from "./missing.gmx"
</script>

<template>
<div>Main</div>
</template>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	_, errors := res.Resolve(file, mainPath)

	// Expect file not found error
	if len(errors) == 0 {
		t.Error("expected file not found error")
	}

	foundNotFound := false
	for _, err := range errors {
		if strings.Contains(err, "failed to read") || strings.Contains(err, "no such file") {
			foundNotFound = true
			break
		}
	}
	if !foundNotFound {
		t.Errorf("expected 'file not found' error, got: %v", errors)
	}
}

func TestMissingMember(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file without sendEmail
	utilsPath := filepath.Join(tmpDir, "utils.gmx")
	utilsContent := `<script>
func otherFunc() error {
  return error("nope")
}
</script>`
	os.WriteFile(utilsPath, []byte(utilsContent), 0644)

	// Try to import sendEmail (doesn't exist)
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import { sendEmail } from "./utils.gmx"
</script>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	_, errors := res.Resolve(file, mainPath)

	// Expect "not found" error
	if len(errors) == 0 {
		t.Error("expected member not found error")
	}

	foundNotFound := false
	for _, err := range errors {
		if strings.Contains(err, "not found") {
			foundNotFound = true
			break
		}
	}
	if !foundNotFound {
		t.Errorf("expected 'not found' error, got: %v", errors)
	}
}

func TestNativeImportsPassThrough(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main with native Go imports
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import "fmt" as fmt
import "time" as time

func logTime() error {
  fmt.Println(time.Now())
  return nil
}
</script>

<template>
<div>Main</div>
</template>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, mainPath)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify native imports are passed through
	hasFmt := false
	hasTime := false
	for _, imp := range resolved.Main.Imports {
		if imp.Path == "fmt" {
			hasFmt = true
		}
		if imp.Path == "time" {
			hasTime = true
		}
	}

	if !hasFmt {
		t.Error("fmt import not passed through")
	}
	if !hasTime {
		t.Error("time import not passed through")
	}
}

func TestNestedComponents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create leaf component
	leafPath := filepath.Join(tmpDir, "Leaf.gmx")
	leafContent := `<script>
model Item {
  id: uuid @pk
  value: string
}
</script>

<template>
<span>{{.Value}}</span>
</template>`
	os.WriteFile(leafPath, []byte(leafContent), 0644)

	// Create container component that uses Leaf
	containerPath := filepath.Join(tmpDir, "Container.gmx")
	containerContent := `<script>
import Leaf from "./Leaf.gmx"
</script>

<template>
<div>Container: {{template "Leaf" .}}</div>
</template>`
	os.WriteFile(containerPath, []byte(containerContent), 0644)

	// Create main that uses Container
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import Container from "./Container.gmx"
</script>

<template>
<div>Main: {{template "Container" .}}</div>
</template>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	resolved, errors := res.Resolve(file, mainPath)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify both components are registered
	if _, ok := resolved.Components["Container"]; !ok {
		t.Error("Container component not found")
	}
	if _, ok := resolved.Components["Leaf"]; !ok {
		t.Error("Leaf component not found (nested)")
	}

	// Verify Item model is merged
	found := false
	for _, m := range resolved.Main.Models {
		if m.Name == "Item" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Item model from Leaf not merged")
	}
}

func TestDefaultImportWithoutTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file without template (not a valid component)
	noTemplatePath := filepath.Join(tmpDir, "notemplate.gmx")
	noTemplateContent := `<script>
model Config {
  id: uuid @pk
}
</script>`
	os.WriteFile(noTemplatePath, []byte(noTemplateContent), 0644)

	// Try to import it as default (should fail)
	mainPath := filepath.Join(tmpDir, "main.gmx")
	mainContent := `<script>
import NoTemplate from "./notemplate.gmx"
</script>

<template>
<div>Main</div>
</template>`
	os.WriteFile(mainPath, []byte(mainContent), 0644)

	file := parseFile(t, mainPath)
	res := New(tmpDir)
	_, errors := res.Resolve(file, mainPath)

	// Expect error about missing template
	if len(errors) == 0 {
		t.Error("expected error about missing template")
	}

	foundTemplateError := false
	for _, err := range errors {
		if strings.Contains(err, "no template") {
			foundTemplateError = true
			break
		}
	}
	if !foundTemplateError {
		t.Errorf("expected 'no template' error, got: %v", errors)
	}
}
