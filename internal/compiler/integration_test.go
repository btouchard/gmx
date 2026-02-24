package compiler

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"gmx/internal/compiler/generator"
	"gmx/internal/compiler/lexer"
	gmxparser "gmx/internal/compiler/parser"
)

// TestFullPipeline tests the complete compilation pipeline
func TestFullPipeline(t *testing.T) {
	input := `<script>
model User {
  id: uuid @pk
  email: string @unique
  posts: Post[]
}

model Post {
  id: uuid @pk
  title: string
  user: User @relation(references: [id])
}

// Go handler code — will be wired in Phase 4
// For now, the generator ignores this block
</script>

<template>
<section id="feed">
  <form hx-post="{{route ` + "`" + `createPost` + "`" + `}}" hx-target="#feed" hx-swap="prepend">
    <input type="text" name="title" class="p-2 border-blue-500" />
    <button type="submit">Publier</button>
  </form>
  {{range .Posts}}
    <div class="card">{{.Title}}</div>
  {{end}}
</section>
</template>

<style>
  .card { padding: 1rem; margin: 0.5rem; background: #f9f9f9; border: 1px solid #eee; }
</style>
`

	// 1. Lex
	l := lexer.New(input)

	// 2. Parse
	p := gmxparser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	if file == nil {
		t.Fatal("ParseGMXFile returned nil")
	}

	// Verify parsed structure
	if len(file.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(file.Models))
	}

	if file.Template == nil {
		t.Fatal("Template block is nil")
	}

	if file.Style == nil {
		t.Fatal("Style block is nil")
	}

	// 3. Generate
	gen := generator.New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 4. Verify generated code is valid Go
	if !isValidScript(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// 5. Verify generated code contains expected elements
	expectedElements := []string{
		"package main",
		"type User struct",
		"type Post struct",
		"ID",
		"Email",
		"Title",
		"Posts []Post",
		"User",
		"gorm:\"primaryKey\"",
		"gorm:\"unique\"",
		"foreignKey:UserID",
		"const pageTemplate",
		"<!DOCTYPE html>",
		"<script src=\"https://cdn.tailwindcss.com\"></script>",
		"<script src=\"https://unpkg.com/htmx.org@2.0.4\"></script>",
		".card { padding: 1rem",
		"{{range .Posts}}",
		"{{.Title}}",
		"var tmpl *template.Template",
		"func init()",
		"template.FuncMap",
		"\"route\":",
		"\"createPost\": \"/api/createPost\"",
		"type PageData struct",
		"CSRFToken string",
		"[]Post",
		"[]User",
		"func handleIndex",
		"func handleCreatePost",
		"db.Find(&data.Posts)",
		"db.Find(&data.Users)",
		"tmpl.Execute(w, data)",
		"func main()",
		"gorm.Open(sqlite.Open(\"gmx.db\")",
		"db.AutoMigrate(&User{}, &Post{})",
		"mux.HandleFunc(\"/\", handleIndex)",
		"mux.HandleFunc(\"/api/createPost\", handleCreatePost)",
		"http.ListenAndServe(\":8080\"",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code missing expected element: %q", expected)
		}
	}
}

// TestMinimalFile tests generating code from a minimal GMX file
func TestMinimalFile(t *testing.T) {
	input := `<script>
model Task {
  id: uuid @pk
  title: string
}
</script>`

	l := lexer.New(input)
	p := gmxparser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	gen := generator.New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidScript(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should have the Task model
	if !strings.Contains(code, "type Task struct") {
		t.Error("Missing Task struct")
	}

	// Should have a working main function even without template
	if !strings.Contains(code, "func main()") {
		t.Error("Missing main function")
	}
}

// TestFileWithTemplateOnly tests a file with only a template (no models)
func TestFileWithTemplateOnly(t *testing.T) {
	input := `<template>
<h1>Hello, World!</h1>
<p>This is a simple page.</p>
</template>

<style>
h1 { color: blue; }
</style>
`

	l := lexer.New(input)
	p := gmxparser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	gen := generator.New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidScript(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should have template
	if !strings.Contains(code, "Hello, World!") {
		t.Error("Missing template content")
	}

	// Should have style
	if !strings.Contains(code, "h1 { color: blue; }") {
		t.Error("Missing style content")
	}

	// Should NOT have GORM imports (no models)
	if strings.Contains(code, "gorm.io/gorm") {
		t.Error("Should not import GORM when there are no models")
	}

	// Should have handleIndex
	if !strings.Contains(code, "func handleIndex") {
		t.Error("Missing handleIndex")
	}
}

// TestComplexRelations tests a file with complex model relationships
func TestComplexRelations(t *testing.T) {
	input := `<script>
model Tenant {
  id: uuid @pk
  name: string
}

model User {
  id: uuid @pk
  email: string @unique
  tenant: Tenant @relation(references: [id])
  posts: Post[]
}

model Post {
  id: uuid @pk
  title: string
  author: User @relation(references: [id])
}
</script>
`

	l := lexer.New(input)
	p := gmxparser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	gen := generator.New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidScript(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Check for all models
	models := []string{"Tenant", "User", "Post"}
	for _, model := range models {
		if !strings.Contains(code, "type "+model+" struct") {
			t.Errorf("Missing %s struct", model)
		}
	}

	// Check for foreign key tags
	if !strings.Contains(code, "foreignKey:TenantID") {
		t.Error("Missing TenantID foreign key")
	}

	if !strings.Contains(code, "foreignKey:AuthorID") {
		t.Error("Missing AuthorID foreign key")
	}

	// Check AutoMigrate includes all models
	if !strings.Contains(code, "db.AutoMigrate(&Tenant{}, &User{}, &Post{})") {
		t.Error("AutoMigrate should include all three models")
	}
}

// isValidScript checks if the given string is valid Go code
func isValidScript(code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", code, parser.AllErrors)
	return err == nil
}

// TestFileWithTopLevelVariables tests a file with top-level let/const declarations
func TestFileWithTopLevelVariables(t *testing.T) {
	input := `<script>
const MAX_RETRIES = 5
const API_VERSION = "v2"
let requestCount: int = 0
let debug: bool = false

model Task {
  id: uuid @pk
  title: string
}

func handle(id: uuid) error {
  return nil
}
</script>`

	l := lexer.New(input)
	p := gmxparser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	// Verify parsed structure
	if len(file.Vars) != 4 {
		t.Errorf("Expected 4 vars, got %d", len(file.Vars))
	}

	if len(file.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(file.Models))
	}

	gen := generator.New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidScript(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Verify generated variables
	expectedVars := []string{
		"const MAX_RETRIES = 5",
		`const API_VERSION = "v2"`,
		"var requestCount int = 0",
		"var debug bool = false",
	}

	for _, expected := range expectedVars {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code missing expected variable: %q", expected)
		}
	}

	// Verify model is still generated
	if !strings.Contains(code, "type Task struct") {
		t.Error("Missing Task struct")
	}

	// Verify function is still generated
	if !strings.Contains(code, "func handle(") {
		t.Error("Missing handle function")
	}

	// Verify variables section comes before models section
	varsIndex := strings.Index(code, "Variables")
	modelsIndex := strings.Index(code, "Models")

	if varsIndex == -1 {
		t.Error("Missing Variables section comment")
	}

	if modelsIndex != -1 && varsIndex > modelsIndex {
		t.Error("Variables section should come before Models section")
	}
}

// TestImportSystemFullPipeline tests the complete compilation pipeline with all three import types
func TestImportSystemFullPipeline(t *testing.T) {
	input := `<script>
import TaskItem from "./components/TaskItem.gmx"
import { sendEmail, MailerConfig } from "./services/mailer.gmx"
import "github.com/stripe/stripe-go" as Stripe

model Task {
  id: uuid @pk
  title: string @min(3) @max(255)
  completed: bool @default(false)
}

service Database {
  provider: "postgres"
  url: string @env("DATABASE_URL")
}

let apiKey: string = "test-api-key"
const maxRetries: int = 3

func createTask(title: string) error {
  let task = Task{
    title: title,
    completed: false
  }
  return error("not implemented")
}
</script>

<template>
<div>
  <h1>Task Manager</h1>
  {{range .Tasks}}
    <div>{{.Title}}</div>
  {{end}}
</div>
</template>

<style scoped>
  h1 { color: blue; }
</style>
`

	// 1. Lex
	l := lexer.New(input)

	// 2. Parse
	p := gmxparser.New(l)
	file := p.ParseGMXFile()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	if file == nil {
		t.Fatal("ParseGMXFile returned nil")
	}

	// Verify imports
	if len(file.Imports) != 3 {
		t.Errorf("Expected 3 imports, got %d", len(file.Imports))
	}

	// Verify default import
	if file.Imports[0].Default != "TaskItem" {
		t.Errorf("Expected default import 'TaskItem', got %q", file.Imports[0].Default)
	}

	// Verify destructured import
	if len(file.Imports[1].Members) != 2 {
		t.Errorf("Expected 2 members in destructured import, got %d", len(file.Imports[1].Members))
	}

	// Verify native Go import
	if !file.Imports[2].IsNative {
		t.Error("Expected third import to be native Go import")
	}
	if file.Imports[2].Alias != "Stripe" {
		t.Errorf("Expected native import alias 'Stripe', got %q", file.Imports[2].Alias)
	}

	// Verify models
	if len(file.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(file.Models))
	}

	// Verify services
	if len(file.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(file.Services))
	}

	// Verify vars
	if len(file.Vars) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(file.Vars))
	}

	// Verify script funcs
	if file.Script == nil || len(file.Script.Funcs) != 1 {
		t.Error("Expected 1 function in script")
	}

	// 3. Generate
	gen := generator.New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 4. Verify generated code is valid Go
	if !isValidScript(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// 5. Verify import generation
	expectedElements := []string{
		"package main",
		"// ========== GMX Imports ==========",
		"// TODO: Component import: TaskItem from ./components/TaskItem.gmx",
		"// TODO: Destructured import: sendEmail, MailerConfig from ./services/mailer.gmx",
		"// Native Go import: github.com/stripe/stripe-go as Stripe",
		`Stripe "github.com/stripe/stripe-go"`,
		"type Task struct",
		"type DatabaseConfig struct",
		"var apiKey string",
		"const maxRetries = 3",
		"func createTask(ctx *GMXContext, title string) error",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code missing expected element: %q", expected)
		}
	}
}
