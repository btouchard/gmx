package shared

import (
	"gmx/internal/compiler/lexer"
	"testing"
)

func TestParseModelDecl(t *testing.T) {
	input := `model Task {
  id:    uuid    @pk @default(uuid_v4)
  title: string  @min(3) @max(255)
  done:  bool    @default(false)
}`

	l := lexer.New(input)
	p := NewParserCore(l)

	// Parser is positioned on 'model' token after NewParserCore
	model := p.ParseModelDecl()

	if model == nil {
		t.Fatal("ParseModelDecl returned nil")
	}

	if model.Name != "Task" {
		t.Errorf("expected model name 'Task', got %q", model.Name)
	}

	if len(model.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(model.Fields))
	}

	// Check first field
	if model.Fields[0].Name != "id" {
		t.Errorf("expected field name 'id', got %q", model.Fields[0].Name)
	}
	if model.Fields[0].Type != "uuid" {
		t.Errorf("expected field type 'uuid', got %q", model.Fields[0].Type)
	}
	if len(model.Fields[0].Annotations) != 2 {
		t.Errorf("expected 2 annotations on id field, got %d", len(model.Fields[0].Annotations))
	}
}

func TestParseServiceDecl(t *testing.T) {
	input := `service Database {
  provider: "postgres"
  url:      string @env("DATABASE_URL")
}`

	l := lexer.New(input)
	p := NewParserCore(l)

	// Parser is positioned on 'service' token after NewParserCore
	svc := p.ParseServiceDecl()

	if svc == nil {
		t.Fatal("ParseServiceDecl returned nil")
	}

	if svc.Name != "Database" {
		t.Errorf("expected service name 'Database', got %q", svc.Name)
	}

	if svc.Provider != "postgres" {
		t.Errorf("expected provider 'postgres', got %q", svc.Provider)
	}

	if len(svc.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(svc.Fields))
	}

	if svc.Fields[0].Name != "url" {
		t.Errorf("expected field name 'url', got %q", svc.Fields[0].Name)
	}
	if svc.Fields[0].Type != "string" {
		t.Errorf("expected field type 'string', got %q", svc.Fields[0].Type)
	}
	if svc.Fields[0].EnvVar != "DATABASE_URL" {
		t.Errorf("expected env var 'DATABASE_URL', got %q", svc.Fields[0].EnvVar)
	}
}

func TestParseServiceWithMethod(t *testing.T) {
	input := `service Mailer {
  provider: "smtp"
  func send(to: string, subject: string) error
}`

	l := lexer.New(input)
	p := NewParserCore(l)

	// Parser is positioned on 'service' token after NewParserCore
	svc := p.ParseServiceDecl()

	if svc == nil {
		t.Fatal("ParseServiceDecl returned nil")
	}

	if len(svc.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(svc.Methods))
	}

	method := svc.Methods[0]
	if method.Name != "send" {
		t.Errorf("expected method name 'send', got %q", method.Name)
	}
	if len(method.Params) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(method.Params))
	}
	if method.ReturnType != "error" {
		t.Errorf("expected return type 'error', got %q", method.ReturnType)
	}
}

func TestParseAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantArgs map[string]string
	}{
		{
			name:     "simple annotation",
			input:    "@pk",
			wantName: "pk",
			wantArgs: map[string]string{},
		},
		{
			name:     "annotation with argument",
			input:    "@default(uuid_v4)",
			wantName: "default",
			wantArgs: map[string]string{"_": "uuid_v4"},
		},
		{
			name:     "annotation with named argument",
			input:    "@relation(references: [id])",
			wantName: "relation",
			wantArgs: map[string]string{"references": "id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := NewParserCore(l)

			ann := p.ParseAnnotation()

			if ann == nil {
				t.Fatal("ParseAnnotation returned nil")
			}

			if ann.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, ann.Name)
			}

			if len(ann.Args) != len(tt.wantArgs) {
				t.Errorf("expected %d args, got %d", len(tt.wantArgs), len(ann.Args))
			}

			for k, v := range tt.wantArgs {
				if ann.Args[k] != v {
					t.Errorf("expected arg %q=%q, got %q", k, v, ann.Args[k])
				}
			}
		})
	}
}

func TestParseModelWithArrayField(t *testing.T) {
	input := `model User {
  posts: Post[]
}`

	l := lexer.New(input)
	p := NewParserCore(l)

	// Parser is positioned on 'model' token after NewParserCore
	model := p.ParseModelDecl()

	if model == nil {
		t.Fatal("ParseModelDecl returned nil")
	}

	if len(model.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(model.Fields))
	}

	if model.Fields[0].Type != "Post[]" {
		t.Errorf("expected field type 'Post[]', got %q", model.Fields[0].Type)
	}
}

func TestParseModelMissingClosingBrace(t *testing.T) {
	input := `model Task {
  id: uuid @pk
`

	l := lexer.New(input)
	p := NewParserCore(l)

	// Parser is positioned on 'model' token after NewParserCore
	model := p.ParseModelDecl()

	// Should return partial model
	if model == nil {
		t.Fatal("expected partial model, got nil")
	}

	if len(p.Errors()) == 0 {
		t.Error("expected error for missing closing brace")
	}
}

func TestParseFieldMissingType(t *testing.T) {
	input := `model Task {
  id: @pk
  name: string
}`

	l := lexer.New(input)
	p := NewParserCore(l)

	// Parser is positioned on 'model' token after NewParserCore
	model := p.ParseModelDecl()

	if model == nil {
		t.Fatal("expected partial model, got nil")
	}

	if len(p.Errors()) == 0 {
		t.Error("expected error for missing type")
	}

	// Should still parse the second field
	if len(model.Fields) < 2 {
		t.Errorf("expected at least 2 fields, got %d", len(model.Fields))
	}
}
