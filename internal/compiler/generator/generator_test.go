package generator

import (
	"fmt"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/utils"
)

func TestGenerateEmptyFile(t *testing.T) {
	file := &ast.GMXFile{
		Models:   nil,
		Script:   nil,
		Template: nil,
		Style:    nil,
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain package main and main function
	if !strings.Contains(code, "package main") {
		t.Error("Generated code missing 'package main'")
	}
	if !strings.Contains(code, "func main()") {
		t.Error("Generated code missing 'func main()'")
	}
}

func TestGenerateWithModels(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
					{
						Name: "email",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "unique", Args: map[string]string{}},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain User struct
	if !strings.Contains(code, "type User struct") {
		t.Error("Generated code missing 'type User struct'")
	}

	// Should contain ID and Email fields
	if !strings.Contains(code, "ID") {
		t.Error("Generated code missing ID field")
	}
	if !strings.Contains(code, "Email") {
		t.Error("Generated code missing Email field")
	}
	if !strings.Contains(code, "string `gorm:") {
		t.Error("Generated code missing string type with GORM tags")
	}

	// Should contain GORM tags
	if !strings.Contains(code, "gorm:\"primaryKey\"") {
		t.Error("Generated code missing GORM primaryKey tag")
	}
	if !strings.Contains(code, "gorm:\"unique\"") {
		t.Error("Generated code missing GORM unique tag")
	}

	// Should import GORM
	if !strings.Contains(code, "gorm.io/gorm") {
		t.Error("Generated code missing GORM import")
	}
}

func TestGenerateWithTemplate(t *testing.T) {
	file := &ast.GMXFile{
		Template: &ast.TemplateBlock{
			Source: "<div>Hello World</div>",
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain template constant
	if !strings.Contains(code, "const pageTemplate") {
		t.Error("Generated code missing pageTemplate constant")
	}

	// Should contain the template content
	if !strings.Contains(code, "Hello World") {
		t.Error("Generated code missing template content")
	}

	// Should contain handleIndex
	if !strings.Contains(code, "func handleIndex") {
		t.Error("Generated code missing handleIndex function")
	}

	// Should import html/template
	if !strings.Contains(code, "html/template") {
		t.Error("Generated code missing html/template import")
	}
}

func TestGenerateWithStyle(t *testing.T) {
	file := &ast.GMXFile{
		Template: &ast.TemplateBlock{
			Source: "<div>Content</div>",
		},
		Style: &ast.StyleBlock{
			Source: ".card { padding: 1rem; }",
			Scoped: false,
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain the style content in the template
	if !strings.Contains(code, ".card { padding: 1rem; }") {
		t.Error("Generated code missing style content")
	}

	// Should have style tag in HTML
	if !strings.Contains(code, "<style>") {
		t.Error("Generated code missing <style> tag")
	}
}

func TestGenerateWithRoutes(t *testing.T) {
	file := &ast.GMXFile{
		Template: &ast.TemplateBlock{
			Source: `<form hx-post="{{route ` + "`" + `createPost` + "`" + `}}">Submit</form>`,
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should have route registry
	if !strings.Contains(code, "\"createPost\": \"/api/createPost\"") {
		t.Error("Generated code missing route registry entry")
	}

	// Should have handler
	if !strings.Contains(code, "func handleCreatePost") {
		t.Error("Generated code missing handleCreatePost function")
	}

	// Should register the route (now uses mux.HandleFunc)
	if !strings.Contains(code, "mux.HandleFunc(\"/api/createPost\"") {
		t.Error("Generated code missing route registration")
	}
}

func TestGenerateFullFile(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
					{
						Name: "email",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "unique", Args: map[string]string{}},
						},
					},
					{
						Name: "posts",
						Type: "Post[]",
					},
				},
			},
			{
				Name: "Post",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
					{
						Name: "title",
						Type: "string",
					},
					{
						Name: "user",
						Type: "User",
						Annotations: []*ast.Annotation{
							{
								Name: "relation",
								Args: map[string]string{"references": "id"},
							},
						},
					},
				},
			},
		},
		Template: &ast.TemplateBlock{
			Source: `<section id="feed">
  <form hx-post="{{route ` + "`" + `createPost` + "`" + `}}" hx-target="#feed" hx-swap="prepend">
    <input type="text" name="title" class="p-2 border-blue-500" />
    <button type="submit">Publier</button>
  </form>
  {{range .Posts}}
    <div class="card">{{.Title}}</div>
  {{end}}
</section>`,
		},
		Style: &ast.StyleBlock{
			Source: ".card { padding: 1rem; margin: 0.5rem; background: #f9f9f9; border: 1px solid #eee; }",
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Check for key components
	checks := []string{
		"package main",
		"type User struct",
		"type Post struct",
		"const pageTemplate",
		"func handleIndex",
		"func handleCreatePost",
		"func main()",
		"gorm.io/gorm",
		"html/template",
		"db.AutoMigrate",
		"mux.HandleFunc", // Changed from http.HandleFunc
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("Generated code missing expected content: %q", check)
		}
	}
}

func TestRelationGeneratesFK(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Post",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
					{
						Name: "user",
						Type: "User",
						Annotations: []*ast.Annotation{
							{
								Name: "relation",
								Args: map[string]string{"references": "id"},
							},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should have User field with FK tag
	if !strings.Contains(code, "User User") {
		t.Error("Generated code missing User relation field")
	}

	// Should have foreignKey tag
	if !strings.Contains(code, "foreignKey:UserID") {
		t.Error("Generated code missing foreignKey tag")
	}
}

func TestEscapeTemplateString(t *testing.T) {
	tests := []struct {
		input    string
		contains []string
	}{
		{
			"Hello World",
			[]string{"`Hello World`"},
		},
		{
			"Hello `World`",
			[]string{"`Hello `", "\"`\"", "` + \"", "`World`"},
		},
	}

	for _, tt := range tests {
		result := escapeTemplateString(tt.input)
		for _, substr := range tt.contains {
			if !strings.Contains(result, substr) {
				t.Errorf("escapeTemplateString(%q) = %q, should contain %q", tt.input, result, substr)
			}
		}
	}
}

func TestMapType(t *testing.T) {
	gen := New()
	tests := []struct {
		input    string
		expected string
	}{
		{"uuid", "string"},
		{"string", "string"},
		{"int", "int"},
		{"float", "float64"},
		{"bool", "bool"},
		{"datetime", "time.Time"},
		{"User", "User"},
		{"Post[]", "[]Post"},
	}

	for _, tt := range tests {
		result := gen.mapType(tt.input)
		if result != tt.expected {
			t.Errorf("mapType(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenRouteRegistry(t *testing.T) {
	gen := New()

	source := `
		<form hx-post="{{route ` + "`" + `createPost` + "`" + `}}">
		<button hx-delete="{{route ` + "`" + `deletePost` + "`" + `}}">
	`

	routes := gen.genRouteRegistry(source)

	expected := map[string]string{
		"createPost": "/api/createPost",
		"deletePost": "/api/deletePost",
	}

	if len(routes) != len(expected) {
		t.Errorf("Expected %d routes, got %d", len(expected), len(routes))
	}

	for name, path := range expected {
		if routes[name] != path {
			t.Errorf("Route %q: expected %q, got %q", name, path, routes[name])
		}
	}
}

func TestGenerateValidation(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
					{
						Name: "title",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "min", Args: map[string]string{"_": "3"}},
							{Name: "max", Args: map[string]string{"_": "255"}},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain Validate method
	if !strings.Contains(code, "func (t *Task) Validate() error") {
		t.Error("Generated code missing Validate method")
	}

	// Should check min length
	if !strings.Contains(code, "if len(t.Title) < 3") {
		t.Error("Generated code missing min length check")
	}

	// Should check max length
	if !strings.Contains(code, "if len(t.Title) > 255") {
		t.Error("Generated code missing max length check")
	}
}

func TestGenerateEmailValidation(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "email",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "email", Args: map[string]string{}},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain Validate method with email check
	if !strings.Contains(code, "func (u *User) Validate() error") {
		t.Error("Generated code missing Validate method")
	}

	if !strings.Contains(code, "!isValidEmail(u.Email)") {
		t.Error("Generated code missing email validation check")
	}

	// Should contain isValidEmail helper
	if !strings.Contains(code, "func isValidEmail(email string) bool") {
		t.Error("Generated code missing isValidEmail helper")
	}

	// Should import regexp
	if !strings.Contains(code, "\"regexp\"") {
		t.Error("Generated code missing regexp import")
	}
}

func TestGenerateBeforeCreate(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Post",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
							{Name: "default", Args: map[string]string{"_": "uuid_v4"}},
						},
					},
					{
						Name: "title",
						Type: "string",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain BeforeCreate hook
	if !strings.Contains(code, "func (p *Post) BeforeCreate(tx *gorm.DB) error") {
		t.Error("Generated code missing BeforeCreate hook")
	}

	// Should set UUID if empty
	if !strings.Contains(code, "if p.ID == \"\"") {
		t.Error("Generated code missing ID empty check")
	}

	if !strings.Contains(code, "p.ID = generateUUID()") {
		t.Error("Generated code missing UUID generation")
	}

	// Should contain generateUUID helper
	if !strings.Contains(code, "func generateUUID() string") {
		t.Error("Generated code missing generateUUID helper")
	}

	// Should import crypto/rand
	if !strings.Contains(code, "\"crypto/rand\"") {
		t.Error("Generated code missing crypto/rand import")
	}
}

func TestGenerateScopedDB(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{
						Name: "tenant_id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "scoped", Args: map[string]string{}},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain scopedDB helper
	if !strings.Contains(code, "func scopedDB(db *gorm.DB, tenantID string) *gorm.DB") {
		t.Error("Generated code missing scopedDB helper")
	}

	if !strings.Contains(code, "db.Where(\"tenant_id = ?\", tenantID)") {
		t.Error("Generated code missing tenant_id filter")
	}
}

func TestGenerateNoValidationIfNotNeeded(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Simple",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
					{
						Name: "name",
						Type: "string",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should NOT contain Validate method
	if strings.Contains(code, "func (s *Simple) Validate() error") {
		t.Error("Generated code should not have Validate method for model without validation")
	}
}

func TestValidationOnIntFields(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Product",
				Fields: []*ast.FieldDecl{
					{
						Name: "price",
						Type: "int",
						Annotations: []*ast.Annotation{
							{Name: "min", Args: map[string]string{"_": "0"}},
							{Name: "max", Args: map[string]string{"_": "10000"}},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should use value comparison, not len()
	if !strings.Contains(code, "if p.Price < 0") {
		t.Error("Generated code should use value comparison for int min")
	}

	if !strings.Contains(code, "if p.Price > 10000") {
		t.Error("Generated code should use value comparison for int max")
	}

	// Should NOT use len() for int fields
	if strings.Contains(code, "len(p.Price)") {
		t.Error("Generated code should not use len() for int fields")
	}
}

func TestGenerateFullFileWithValidation(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
							{Name: "default", Args: map[string]string{"_": "uuid_v4"}},
						},
					},
					{
						Name: "email",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "unique", Args: map[string]string{}},
							{Name: "email", Args: map[string]string{}},
						},
					},
					{
						Name: "name",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "min", Args: map[string]string{"_": "2"}},
							{Name: "max", Args: map[string]string{"_": "100"}},
						},
					},
				},
			},
			{
				Name: "Post",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
							{Name: "default", Args: map[string]string{"_": "uuid_v4"}},
						},
					},
					{
						Name: "title",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "min", Args: map[string]string{"_": "3"}},
							{Name: "max", Args: map[string]string{"_": "255"}},
						},
					},
					{
						Name: "tenant_id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "scoped", Args: map[string]string{}},
						},
					},
				},
			},
		},
		Template: &ast.TemplateBlock{
			Source: `<div>Test</div>`,
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Check for all features
	checks := []string{
		"func generateUUID() string",
		"func isValidEmail(email string) bool",
		"func scopedDB(db *gorm.DB, tenantID string) *gorm.DB",
		"func (u *User) Validate() error",
		"func (p *Post) Validate() error",
		"func (u *User) BeforeCreate(tx *gorm.DB) error",
		"func (p *Post) BeforeCreate(tx *gorm.DB) error",
		"\"crypto/rand\"",
		"\"regexp\"",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("Generated code missing expected content: %q", check)
		}
	}
}

func TestConditionalImports(t *testing.T) {
	// Test file without UUID or email
	fileNoSpecial := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "Simple",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "int"},
					{Name: "name", Type: "string"},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(fileNoSpecial)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// crypto/rand is now always imported for CSRF token generation
	if !strings.Contains(code, "\"crypto/rand\"") {
		t.Error("Generated code should always import crypto/rand for CSRF protection")
	}

	if strings.Contains(code, "\"regexp\"") {
		t.Error("Generated code should not import regexp when not needed")
	}

	// Test file with UUID
	fileWithUUID := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "default", Args: map[string]string{"_": "uuid_v4"}},
						},
					},
				},
			},
		},
	}

	code, err = gen.Generate(fileWithUUID)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(code, "\"crypto/rand\"") {
		t.Error("Generated code should import crypto/rand when UUID is used")
	}

	// Test file with email
	fileWithEmail := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "email",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "email", Args: map[string]string{}},
						},
					},
				},
			},
		},
	}

	code, err = gen.Generate(fileWithEmail)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(code, "\"regexp\"") {
		t.Error("Generated code should import regexp when @email is used")
	}
}

func TestSecurityHeaders(t *testing.T) {
	gen := New()

	file := &ast.GMXFile{
		Template: &ast.TemplateBlock{
			Source: `<div>Test</div>`,
		},
	}

	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should generate securityHeaders middleware
	if !strings.Contains(code, "func securityHeaders") {
		t.Error("Generated code should contain securityHeaders middleware")
	}

	// Should wrap mux with securityHeaders
	if !strings.Contains(code, "securityHeaders(mux)") {
		t.Error("Generated code should wrap mux with securityHeaders")
	}

	// Should set security headers
	checks := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("Generated code should set security header: %q", check)
		}
	}
}

func TestHTTPMethodGuard(t *testing.T) {
	gen := New()

	file := &ast.GMXFile{
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name: "createTask",
					Params: []*ast.Param{
						{Name: "title", Type: "string"},
					},
					Body: []ast.Statement{},
				},
				{
					Name: "deleteTask",
					Params: []*ast.Param{
						{Name: "id", Type: "uuid"},
					},
					Body: []ast.Statement{},
				},
				{
					Name: "getTask",
					Params: []*ast.Param{
						{Name: "id", Type: "uuid"},
					},
					Body: []ast.Statement{},
				},
			},
		},
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk", Args: map[string]string{}}}},
				},
			},
		},
	}

	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Debug: print if handlers are generated
	if !strings.Contains(code, "handleCreateTask") {
		t.Log("WARNING: handleCreateTask not found in generated code")
	}

	// createTask should check for POST
	if !strings.Contains(code, "r.Method != http.MethodPost") {
		t.Errorf("createTask handler should guard for POST method\nGenerated code contains: %v", strings.Contains(code, "handleCreateTask"))
	}

	// deleteTask should check for DELETE
	if !strings.Contains(code, "r.Method != http.MethodDelete") {
		t.Errorf("deleteTask handler should guard for DELETE method\nGenerated code contains: %v", strings.Contains(code, "handleDeleteTask"))
	}

	// getTask should check for GET
	if !strings.Contains(code, "r.Method != http.MethodGet") {
		t.Errorf("getTask handler should guard for GET method\nGenerated code contains: %v", strings.Contains(code, "handleGetTask"))
	}
}

func TestUUIDValidation(t *testing.T) {
	gen := New()

	file := &ast.GMXFile{
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name: "toggleTask",
					Params: []*ast.Param{
						{Name: "id", Type: "uuid"},
					},
					Body: []ast.Statement{},
				},
			},
		},
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk", Args: map[string]string{}}}},
				},
			},
		},
	}

	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should generate isValidUUID helper
	if !strings.Contains(code, "func isValidUUID") {
		t.Error("Generated code should contain isValidUUID helper")
	}

	// Handler should validate UUID
	if !strings.Contains(code, "isValidUUID(id)") {
		t.Error("Handler should validate UUID parameter")
	}

	// Should return BadRequest on invalid UUID
	if !strings.Contains(code, "Invalid ID format") {
		t.Error("Handler should return 'Invalid ID format' error")
	}
}

func TestSecureErrorHandling(t *testing.T) {
	gen := New()

	file := &ast.GMXFile{
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name: "createTask",
					Params: []*ast.Param{
						{Name: "title", Type: "string"},
					},
					Body: []ast.Statement{},
				},
			},
		},
		Template: &ast.TemplateBlock{
			Source: `<div>{{.}}</div>`,
		},
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk", Args: map[string]string{}}}},
				},
			},
		},
	}

	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should log errors instead of exposing them
	if !strings.Contains(code, "log.Printf(\"handler error: %v\", err)") {
		t.Error("Handler should log errors server-side")
	}

	// Should return generic error message
	if !strings.Contains(code, "Internal Server Error") {
		t.Error("Handler should return generic 'Internal Server Error' message")
	}

	// Template errors should also be logged
	if !strings.Contains(code, "log.Printf(\"template error: %v\", err)") {
		t.Error("Template handler should log errors")
	}

	// Should NOT expose err.Error() in HTTP response (script handlers)
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.Contains(line, "http.Error(w, err.Error()") && strings.Contains(line, "handleCreate") {
			t.Errorf("Line %d exposes internal error to client: %s", i+1, line)
		}
	}
}

func TestParameterValidation(t *testing.T) {
	gen := New()

	file := &ast.GMXFile{
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name: "createTask",
					Params: []*ast.Param{
						{Name: "title", Type: "string"},
						{Name: "priority", Type: "int"},
					},
					Body: []ast.Statement{},
				},
			},
		},
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk", Args: map[string]string{}}}},
				},
			},
		},
	}

	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should check for empty parameters
	if !strings.Contains(code, "Missing required parameter") {
		t.Error("Handler should validate non-empty parameters")
	}

	// Should validate parameter before using it
	requiredChecks := []string{
		`if title == ""`,
		`if priority == ""`,
	}

	for _, check := range requiredChecks {
		if !strings.Contains(code, check) {
			t.Errorf("Handler should check for empty parameter: %s", check)
		}
	}
}

// Helper function to validate Go syntax
func isValidGo(code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", code, parser.AllErrors)
	return err == nil
}

// ========== SERVICE TESTS ==========

func TestGenServiceConfig(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Database",
				Provider: "postgres",
				Fields: []*ast.ServiceField{
					{
						Name:   "url",
						Type:   "string",
						EnvVar: "DATABASE_URL",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Debug: print the generated code section
	if !strings.Contains(code, "DatabaseConfig") {
		t.Logf("Generated code:\n%s", code)
	}

	// Should generate config struct
	if !strings.Contains(code, "type DatabaseConfig struct") {
		t.Error("Generated code missing DatabaseConfig struct")
	}

	if !strings.Contains(code, "Provider string") {
		t.Error("Generated code missing Provider field")
	}

	if !strings.Contains(code, "Url") || !strings.Contains(code, "type DatabaseConfig struct") {
		t.Errorf("Generated code missing Url field (PascalCase of 'url')")
	}
}

func TestGenServiceInit(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Mailer",
				Provider: "smtp",
				Fields: []*ast.ServiceField{
					{
						Name:   "host",
						Type:   "string",
						EnvVar: "SMTP_HOST",
					},
					{
						Name:   "pass",
						Type:   "string",
						EnvVar: "SMTP_PASS",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate init function
	if !strings.Contains(code, "func initMailer() *MailerConfig") {
		t.Error("Generated code missing initMailer function")
	}

	// Should read env vars
	if !strings.Contains(code, `os.Getenv("SMTP_HOST")`) {
		t.Error("Generated code missing SMTP_HOST env read")
	}

	if !strings.Contains(code, `os.Getenv("SMTP_PASS")`) {
		t.Error("Generated code missing SMTP_PASS env read")
	}

	// Should check for missing env vars
	if !strings.Contains(code, `log.Fatal("missing required env var: SMTP_HOST")`) {
		t.Error("Generated code missing env var validation")
	}

	// Should import os
	if !strings.Contains(code, `"os"`) {
		t.Error("Generated code missing os import")
	}
}

func TestGenServiceInterface(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Mailer",
				Provider: "smtp",
				Methods: []*ast.ServiceMethod{
					{
						Name: "send",
						Params: []*ast.Param{
							{Name: "to", Type: "string"},
							{Name: "subject", Type: "string"},
							{Name: "body", Type: "string"},
						},
						ReturnType: "error",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate interface
	if !strings.Contains(code, "type MailerService interface") {
		t.Error("Generated code missing MailerService interface")
	}

	// Should have Send method (PascalCase)
	if !strings.Contains(code, "Send(to string, subject string, body string) error") {
		t.Error("Generated code missing Send method signature")
	}
}

func TestGenServiceStub(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Storage",
				Provider: "s3",
				Methods: []*ast.ServiceMethod{
					{
						Name: "upload",
						Params: []*ast.Param{
							{Name: "key", Type: "string"},
							{Name: "data", Type: "string"},
						},
						ReturnType: "error",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate stub struct
	if !strings.Contains(code, "type storageStub struct") {
		t.Error("Generated code missing storageStub struct")
	}

	// Should implement Upload method
	if !strings.Contains(code, "func (s *storageStub) Upload(key string, data string) error") {
		t.Error("Generated code missing Upload stub method")
	}

	// Should log method call
	if !strings.Contains(code, `log.Printf("[%s] Storage.Upload called (stub)"`) {
		t.Error("Generated code missing log statement in stub")
	}

	// Should generate factory function
	if !strings.Contains(code, "func newStorageService(cfg *StorageConfig) StorageService") {
		t.Error("Generated code missing newStorageService factory")
	}

	if !strings.Contains(code, "return &storageStub{config: cfg}") {
		t.Error("Generated code missing stub instantiation")
	}
}

func TestGenServiceImportOs(t *testing.T) {
	fileWithEnv := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Cache",
				Provider: "redis",
				Fields: []*ast.ServiceField{
					{
						Name:   "host",
						Type:   "string",
						EnvVar: "REDIS_HOST",
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(fileWithEnv)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should import os when @env is used
	if !strings.Contains(code, `"os"`) {
		t.Error("Generated code should import os when services use @env")
	}

	// File without env should NOT import os
	fileWithoutEnv := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Logger",
				Provider: "console",
			},
		},
	}

	code2, err := gen.Generate(fileWithoutEnv)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should NOT import os when no @env
	if strings.Contains(code2, `"os"`) {
		t.Error("Generated code should not import os when services don't use @env")
	}
}

func TestServiceMainInitialization(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Database",
				Provider: "sqlite",
				Fields: []*ast.ServiceField{
					{
						Name:   "url",
						Type:   "string",
						EnvVar: "DATABASE_URL",
					},
				},
			},
			{
				Name:     "Mailer",
				Provider: "smtp",
				Methods: []*ast.ServiceMethod{
					{
						Name:       "send",
						Params:     []*ast.Param{{Name: "to", Type: "string"}},
						ReturnType: "error",
					},
				},
			},
		},
		Template: &ast.TemplateBlock{
			Source: "<div>Test</div>",
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should initialize Database config
	if !strings.Contains(code, "databaseCfg := initDatabase()") {
		t.Error("Generated code missing Database config initialization")
	}

	// Should initialize Mailer config and service
	if !strings.Contains(code, "mailerCfg := initMailer()") {
		t.Error("Generated code missing Mailer config initialization")
	}

	if !strings.Contains(code, "mailerSvc := newMailerService(mailerCfg)") {
		t.Error("Generated code missing Mailer service initialization")
	}

	// Should suppress unused var warnings
	if !strings.Contains(code, "_ = databaseCfg") {
		t.Error("Generated code missing databaseCfg unused suppression")
	}

	if !strings.Contains(code, "_ = mailerSvc") {
		t.Error("Generated code missing mailerSvc unused suppression")
	}
}

// Test zeroValue function (0% coverage)
func TestZeroValue(t *testing.T) {
	gen := New()

	tests := []struct {
		typ      string
		expected string
	}{
		{"string", "\"\""},
		{"int", "0"},
		{"bool", "false"},
		{"error", "nil"},
		{"float64", "nil"},  // default case
		{"*User", "nil"},    // default case
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			result := gen.zeroValue(tt.typ)
			if result != tt.expected {
				t.Errorf("zeroValue(%q) = %q, want %q", tt.typ, result, tt.expected)
			}
		})
	}
}

// Test genHTTPClient function (0% coverage)
func TestGenHTTPClient(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "PaymentAPI",
				Provider: "http",
				Fields: []*ast.ServiceField{
					{Name: "baseUrl", Type: "string", EnvVar: "PAYMENT_URL"},
					{Name: "apiKey", Type: "string", EnvVar: "API_KEY"},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for HTTP client struct
	if !strings.Contains(result, "type PaymentAPIClient struct") {
		t.Error("Generated code missing PaymentAPIClient struct")
	}

	// Check for http.Client field
	if !strings.Contains(result, "http   *http.Client") {
		t.Error("Generated code missing http.Client field")
	}

	// Check for factory function
	if !strings.Contains(result, "func newPaymentAPIClient(cfg *PaymentAPIConfig) *PaymentAPIClient") {
		t.Error("Generated code missing factory function")
	}

	// Check for GET method
	if !strings.Contains(result, "func (c *PaymentAPIClient) Get(path string) (*http.Response, error)") {
		t.Error("Generated code missing Get method")
	}

	// Check for POST method
	if !strings.Contains(result, "func (c *PaymentAPIClient) Post(path string, body io.Reader) (*http.Response, error)") {
		t.Error("Generated code missing Post method")
	}

	// Check for Authorization header with Bearer token
	if !strings.Contains(result, `req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)`) {
		t.Error("Generated code missing Bearer token authorization")
	}
}

// Test genHTTPClient without apiKey field
func TestGenHTTPClientWithoutApiKey(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "PublicAPI",
				Provider: "http",
				Fields: []*ast.ServiceField{
					{Name: "baseUrl", Type: "string", EnvVar: "PUBLIC_URL"},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that client is generated
	if !strings.Contains(result, "type PublicAPIClient struct") {
		t.Error("Generated code missing PublicAPIClient struct")
	}

	// Check that Authorization header code is not present (no apiKey field)
	authCount := strings.Count(result, `"Authorization"`)
	if authCount > 0 {
		t.Error("Generated code should not include Authorization header without apiKey field")
	}
}

// Test genSMTPImpl with all fields
func TestGenSMTPImplComplete(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Mailer",
				Provider: "smtp",
				Fields: []*ast.ServiceField{
					{Name: "host", Type: "string", EnvVar: "SMTP_HOST"},
					{Name: "port", Type: "string", EnvVar: "SMTP_PORT"},
					{Name: "user", Type: "string", EnvVar: "SMTP_USER"},
					{Name: "password", Type: "string", EnvVar: "SMTP_PASS"},
					{Name: "from", Type: "string", EnvVar: "SMTP_FROM"},
				},
				Methods: []*ast.ServiceMethod{
					{
						Name:       "send",
						Params:     []*ast.Param{{Name: "to", Type: "string"}, {Name: "subject", Type: "string"}, {Name: "body", Type: "string"}},
						ReturnType: "error",
					},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for SMTP implementation with all fields
	// Port is now a string field, so it uses simple string concatenation
	if !strings.Contains(result, `m.config.Host + ":" + m.config.Port`) {
		t.Error("Expected port handling with string concatenation")
	}

	if !strings.Contains(result, "smtp.PlainAuth") {
		t.Error("Expected PlainAuth with user/password")
	}
}

// Test genSMTPImpl without port (uses default)
func TestGenSMTPImplWithoutPort(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "Mailer",
				Provider: "smtp",
				Fields: []*ast.ServiceField{
					{Name: "host", Type: "string", EnvVar: "SMTP_HOST"},
					{Name: "user", Type: "string", EnvVar: "SMTP_USER"},
					{Name: "password", Type: "string", EnvVar: "SMTP_PASS"},
				},
				Methods: []*ast.ServiceMethod{
					{
						Name:       "send",
						Params:     []*ast.Param{{Name: "to", Type: "string"}},
						ReturnType: "error",
					},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should use host directly without port
	if !strings.Contains(result, "m.config.Host") {
		t.Error("Expected direct host usage")
	}
}

// Test handler HTTP method inference
func TestInferHTTPMethod(t *testing.T) {
	tests := []struct {
		handlerName string
		wantMethod  string
	}{
		{"getTasks", "Get"},
		{"createTask", "Post"},
		{"addTask", "Post"},
		{"updateTask", "Patch"}, // update -> PATCH
		{"editTask", "Patch"},   // edit -> PATCH
		{"toggleTask", "Patch"}, // toggle -> PATCH
		{"deleteTask", "Delete"},
		{"removeTask", "Delete"},
		{"listTasks", "Get"},
		{"findTask", "Get"},
		{"customHandler", "Post"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.handlerName, func(t *testing.T) {
			file := &ast.GMXFile{
				Script: &ast.ScriptBlock{
					Funcs: []*ast.FuncDecl{
						{
							Name:   tt.handlerName,
							Params: []*ast.Param{},
							Body:   []ast.Statement{},
						},
					},
				},
			}

			gen := New()
			result, err := gen.Generate(file)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Check that handler function is generated with correct name
			handlerFuncName := "handle" + utils.Capitalize(tt.handlerName)
			if !strings.Contains(result, fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request)", handlerFuncName)) {
				t.Errorf("Handler function %s not found in output", handlerFuncName)
			}

			// Check that HTTP method guard uses correct method
			// Convert method name to Go constant format: GET -> Get, POST -> Post, DELETE -> Delete
			methodConstant := utils.Capitalize(strings.ToLower(tt.wantMethod))
			expectedGuard := fmt.Sprintf("if r.Method != http.Method%s", methodConstant)
			if !strings.Contains(result, expectedGuard) {
				t.Errorf("Expected method guard for %s (http.Method%s), got none", tt.wantMethod, methodConstant)
			}
		})
	}
}

// Test template generation without </head> tag
func TestGenTemplateWithoutHead(t *testing.T) {
	file := &ast.GMXFile{
		Template: &ast.TemplateBlock{
			Source: "<div>Simple template without head</div>",
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should still generate valid template code
	// Template constant is now called pageTemplate
	if !strings.Contains(result, "const pageTemplate = ") {
		t.Error("Expected pageTemplate constant")
	}
}

// Test template with HTML wrapper (has </head>)
func TestGenTemplateWithHead(t *testing.T) {
	file := &ast.GMXFile{
		Template: &ast.TemplateBlock{
			Source: "<html><head><title>Test</title></head><body><div>Content</div></body></html>",
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(result, "</head>") {
		t.Error("Expected head tag in template")
	}
}

// Test generation with postgres database
func TestGenMainWithPostgres(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk"}}},
					{Name: "name", Type: "string"},
				},
			},
		},
		Services: []*ast.ServiceDecl{
			{
				Name:     "Database",
				Provider: "postgres",
				Fields: []*ast.ServiceField{
					{Name: "url", Type: "string", EnvVar: "DATABASE_URL"},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for GORM postgres driver import (only imported when models exist)
	if !strings.Contains(result, `"gorm.io/driver/postgres"`) {
		t.Error("Expected GORM postgres driver import")
	}
}

// Test generation with sqlite database (default)
func TestGenMainWithSqlite(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk"}}},
					{Name: "name", Type: "string"},
				},
			},
		},
		Services: []*ast.ServiceDecl{
			{
				Name:     "Database",
				Provider: "sqlite",
				Fields: []*ast.ServiceField{
					{Name: "url", Type: "string", EnvVar: "DATABASE_URL"},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should generate database connection code (only when models exist)
	if !strings.Contains(result, "gorm.Open") {
		t.Error("Expected GORM open call")
	}

	// Should import sqlite driver
	if !strings.Contains(result, `"gorm.io/driver/sqlite"`) {
		t.Error("Expected GORM sqlite driver import")
	}
}

// Test import generation with HTTP service (requires io and time)
func TestGenImportsWithHTTPService(t *testing.T) {
	file := &ast.GMXFile{
		Services: []*ast.ServiceDecl{
			{
				Name:     "API",
				Provider: "http",
				Fields: []*ast.ServiceField{
					{Name: "baseUrl", Type: "string", EnvVar: "API_URL"},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for io and time imports (needed for HTTP client)
	if !strings.Contains(result, `"io"`) {
		t.Error("Expected io import for HTTP client")
	}
	if !strings.Contains(result, `"time"`) {
		t.Error("Expected time import for HTTP client")
	}
}

// Test validation generation with different types
func TestGenValidationTypes(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{
						Name: "email",
						Type: "string",
						Annotations: []*ast.Annotation{
							{Name: "required", Args: map[string]string{}},
						},
					},
					{
						Name: "age",
						Type: "int",
						Annotations: []*ast.Annotation{
							{Name: "min", Args: map[string]string{"value": "18"}},
						},
					},
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
				},
			},
		},
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name: "getUser",
					Params: []*ast.Param{
						{Name: "id", Type: "uuid"},
						{Name: "age", Type: "int"},
					},
					Body: []ast.Statement{},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for strconv import (needed for int parameter parsing in script handlers)
	if !strings.Contains(result, `"strconv"`) {
		t.Error("Expected strconv import for int parameter parsing")
	}

	// Check for UUID validation (needed for uuid pk field)
	if !strings.Contains(result, "isValidUUID") {
		t.Error("Expected UUID validation helper")
	}
}

// Test handler with different parameter types
func TestGenHandlersWithDifferentParamTypes(t *testing.T) {
	file := &ast.GMXFile{
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name: "getTask",
					Params: []*ast.Param{
						{Name: "id", Type: "uuid"},
						{Name: "includeDetails", Type: "bool"},
					},
					Body: []ast.Statement{},
				},
			},
		},
	}

	gen := New()
	result, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for parameter extraction (now uses PathValue with FormValue fallback)
	if !strings.Contains(result, `r.PathValue("id")`) {
		t.Error("Expected id parameter extraction using PathValue")
	}
	if !strings.Contains(result, `r.PathValue("includeDetails")`) {
		t.Error("Expected includeDetails parameter extraction using PathValue")
	}
	// Should also have FormValue fallback
	if !strings.Contains(result, `r.FormValue("id")`) {
		t.Error("Expected FormValue fallback for id")
	}
}

// ========== VARIABLE GENERATION TESTS ==========

func TestGenVarsWithConst(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "MAX_RETRIES",
				Type:    "",
				Value:   &ast.IntLit{Value: "5"},
				IsConst: true,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate const declaration
	if !strings.Contains(code, "const MAX_RETRIES = 5") {
		t.Error("Generated code missing const MAX_RETRIES = 5")
	}
}

func TestGenVarsWithLetAndExplicitType(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "requestCount",
				Type:    "int",
				Value:   &ast.IntLit{Value: "0"},
				IsConst: false,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate var declaration with explicit type
	if !strings.Contains(code, "var requestCount int = 0") {
		t.Error("Generated code missing var requestCount int = 0")
	}
}

func TestGenVarsWithInferredType(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "debug",
				Type:    "",
				Value:   &ast.BoolLit{Value: false},
				IsConst: false,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate var declaration with inferred type or explicit bool type
	if !strings.Contains(code, "var debug = false") && !strings.Contains(code, "var debug bool = false") {
		t.Errorf("Generated code missing var debug declaration, got:\n%s", code)
	}
}

func TestGenVarsWithStringConst(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "API_VERSION",
				Type:    "",
				Value:   &ast.StringLit{Value: "v2"},
				IsConst: true,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate const declaration with quoted string
	if !strings.Contains(code, `const API_VERSION = "v2"`) {
		t.Error(`Generated code missing const API_VERSION = "v2"`)
	}
}

func TestGenMultipleVars(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "MAX_RETRIES",
				Type:    "",
				Value:   &ast.IntLit{Value: "5"},
				IsConst: true,
			},
			{
				Name:    "API_VERSION",
				Type:    "",
				Value:   &ast.StringLit{Value: "v2"},
				IsConst: true,
			},
			{
				Name:    "requestCount",
				Type:    "int",
				Value:   &ast.IntLit{Value: "0"},
				IsConst: false,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should have all three variables
	if !strings.Contains(code, "const MAX_RETRIES = 5") {
		t.Error("Generated code missing const MAX_RETRIES = 5")
	}
	if !strings.Contains(code, `const API_VERSION = "v2"`) {
		t.Error(`Generated code missing const API_VERSION = "v2"`)
	}
	if !strings.Contains(code, "var requestCount int = 0") {
		t.Error("Generated code missing var requestCount int = 0")
	}
}

func TestGenVarsWithModelsAndFuncs(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "MAX_RETRIES",
				Type:    "",
				Value:   &ast.IntLit{Value: "5"},
				IsConst: true,
			},
		},
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk"}}},
				},
			},
		},
		Script: &ast.ScriptBlock{
			Funcs: []*ast.FuncDecl{
				{
					Name:   "getTask",
					Params: []*ast.Param{{Name: "id", Type: "uuid"}},
					Body:   []ast.Statement{},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should have variables section before models
	varsIndex := strings.Index(code, "Variables")
	modelsIndex := strings.Index(code, "Models")

	if varsIndex == -1 {
		t.Error("Generated code missing Variables section")
	}

	if modelsIndex != -1 && varsIndex > modelsIndex {
		t.Error("Variables section should come before Models section")
	}

	// Should contain variable
	if !strings.Contains(code, "const MAX_RETRIES = 5") {
		t.Error("Generated code missing const MAX_RETRIES = 5")
	}

	// Should contain model
	if !strings.Contains(code, "type Task struct") {
		t.Error("Generated code missing Task model")
	}

	// Should contain function
	if !strings.Contains(code, "func getTask(") {
		t.Error("Generated code missing getTask function")
	}
}

func TestGenVarsWithFloatType(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name:    "pi",
				Type:    "float",
				Value:   &ast.FloatLit{Value: "3.14"},
				IsConst: false,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should map float to float64
	if !strings.Contains(code, "var pi float64 = 3.14") {
		t.Error("Generated code missing var pi float64 = 3.14")
	}
}

func TestGenVarsWithBinaryExpr(t *testing.T) {
	file := &ast.GMXFile{
		Vars: []*ast.VarDecl{
			{
				Name: "TOTAL",
				Type: "",
				Value: &ast.BinaryExpr{
					Left:  &ast.IntLit{Value: "10"},
					Op:    "+",
					Right: &ast.IntLit{Value: "20"},
				},
				IsConst: true,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should generate const with binary expression
	if !strings.Contains(code, "const TOTAL = 10 + 20") {
		t.Error("Generated code missing const TOTAL = 10 + 20")
	}
}

// ============ IMPORT TESTS ============

func TestGenerateNativeGoImport(t *testing.T) {
	file := &ast.GMXFile{
		Imports: []*ast.ImportDecl{
			{
				Path:     "github.com/stripe/stripe-go",
				Alias:    "Stripe",
				IsNative: true,
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain the native Go import with alias
	if !strings.Contains(code, `Stripe "github.com/stripe/stripe-go"`) {
		t.Error("Generated code missing native Go import with alias")
	}

	// Should contain GMX import comment
	if !strings.Contains(code, "// Native Go import: github.com/stripe/stripe-go as Stripe") {
		t.Error("Generated code missing GMX import comment for native import")
	}
}

func TestGenerateDefaultImportPlaceholder(t *testing.T) {
	file := &ast.GMXFile{
		Imports: []*ast.ImportDecl{
			{
				Default: "TaskItem",
				Path:    "./components/TaskItem.gmx",
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain TODO placeholder comment
	if !strings.Contains(code, "// TODO: Component import: TaskItem from ./components/TaskItem.gmx") {
		t.Error("Generated code missing TODO placeholder for default import")
	}
}

func TestGenerateDestructuredImportPlaceholder(t *testing.T) {
	file := &ast.GMXFile{
		Imports: []*ast.ImportDecl{
			{
				Members: []string{"sendEmail", "MailerConfig"},
				Path:    "./services/mailer.gmx",
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain TODO placeholder comment
	if !strings.Contains(code, "// TODO: Destructured import: sendEmail, MailerConfig from ./services/mailer.gmx") {
		t.Error("Generated code missing TODO placeholder for destructured import")
	}
}

func TestGenerateMixedImports(t *testing.T) {
	file := &ast.GMXFile{
		Imports: []*ast.ImportDecl{
			{
				Default: "TaskItem",
				Path:    "./components/TaskItem.gmx",
			},
			{
				Members: []string{"sendEmail"},
				Path:    "./services/mailer.gmx",
			},
			{
				Path:     "github.com/stripe/stripe-go",
				Alias:    "Stripe",
				IsNative: true,
			},
		},
		Models: []*ast.ModelDecl{
			{
				Name: "Task",
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "uuid",
						Annotations: []*ast.Annotation{
							{Name: "pk", Args: map[string]string{}},
						},
					},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should produce valid Go code
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Should contain all import comments
	if !strings.Contains(code, "// TODO: Component import: TaskItem from ./components/TaskItem.gmx") {
		t.Error("Generated code missing default import placeholder")
	}
	if !strings.Contains(code, "// TODO: Destructured import: sendEmail from ./services/mailer.gmx") {
		t.Error("Generated code missing destructured import placeholder")
	}
	if !strings.Contains(code, "// Native Go import: github.com/stripe/stripe-go as Stripe") {
		t.Error("Generated code missing native import comment")
	}

	// Should contain the actual native Go import
	if !strings.Contains(code, `Stripe "github.com/stripe/stripe-go"`) {
		t.Error("Generated code missing native Go import in import block")
	}

	// Should still contain model generation
	if !strings.Contains(code, "type Task struct") {
		t.Error("Generated code missing Task model")
	}
}
