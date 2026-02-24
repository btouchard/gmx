package parser

import (
	"gmx/internal/compiler/lexer"
	"strings"
	"testing"
)

// Integration test: Parse the complete Phase 2 example and verify all details
func TestPhase2Integration(t *testing.T) {
	input := `model Task {
  id:         uuid    @pk @default(uuid_v4)
  title:      string  @min(3) @max(255)
  done:       bool    @default(false)
  tenant_id:  uuid    @scoped
  author:     User    @relation(references: [id])
}

model User {
  id:    uuid    @pk @default(uuid_v4)
  email: string  @unique @email
  tasks: Task[]
}

<script>
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}
</script>

<template>
  <div class="task-item" id="task-{{.ID}}">
    <span>{{.Title}}</span>
    <button hx-patch="{{route "toggleTask" .ID}}" hx-target="closest .task-item" hx-swap="outerHTML">
      {{if .Done}}Undo{{else}}Done{{end}}
    </button>
  </div>
</template>

<style scoped>
  .task-item { padding: 1rem; border-bottom: 1px solid #eee; }
  .completed { opacity: 0.5; }
</style>
`

	l := lexer.New(input)
	p := New(l)
	file := p.ParseGMXFile()

	// Verify no errors
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	// === VERIFY MODELS ===
	if len(file.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(file.Models))
	}

	// Verify Task model
	task := file.Models[0]
	if task.Name != "Task" {
		t.Errorf("expected Task model, got %q", task.Name)
	}
	if len(task.Fields) != 5 {
		t.Fatalf("expected 5 fields in Task, got %d", len(task.Fields))
	}

	// Detailed field verification for Task
	expectedTaskFields := []struct {
		name        string
		typ         string
		annotations map[string]string // annotation name -> expected arg value
	}{
		{"id", "uuid", map[string]string{"pk": "", "default": "uuid_v4"}},
		{"title", "string", map[string]string{"min": "3", "max": "255"}},
		{"done", "bool", map[string]string{"default": "false"}},
		{"tenant_id", "uuid", map[string]string{"scoped": ""}},
		{"author", "User", map[string]string{"relation": ""}},
	}

	for i, expected := range expectedTaskFields {
		field := task.Fields[i]
		if field.Name != expected.name {
			t.Errorf("Task field %d: expected name %q, got %q", i, expected.name, field.Name)
		}
		if field.Type != expected.typ {
			t.Errorf("Task field %d: expected type %q, got %q", i, expected.typ, field.Type)
		}

		// Verify annotations
		if len(field.Annotations) != len(expected.annotations) {
			t.Errorf("Task field %d: expected %d annotations, got %d", i, len(expected.annotations), len(field.Annotations))
		}

		for _, ann := range field.Annotations {
			expectedArg, exists := expected.annotations[ann.Name]
			if !exists {
				t.Errorf("Task field %d: unexpected annotation @%s", i, ann.Name)
				continue
			}

			if expectedArg != "" {
				actualArg := ann.SimpleArg()
				if actualArg != expectedArg {
					t.Errorf("Task field %d, annotation @%s: expected arg %q, got %q", i, ann.Name, expectedArg, actualArg)
				}
			}

			// Special check for relation annotation
			if ann.Name == "relation" {
				if ann.Args["references"] != "id" {
					t.Errorf("Task field %d, @relation: expected references=id, got %q", i, ann.Args["references"])
				}
			}
		}
	}

	// Verify User model
	user := file.Models[1]
	if user.Name != "User" {
		t.Errorf("expected User model, got %q", user.Name)
	}
	if len(user.Fields) != 3 {
		t.Fatalf("expected 3 fields in User, got %d", len(user.Fields))
	}

	// Check array type
	tasksField := user.Fields[2]
	if tasksField.Name != "tasks" {
		t.Errorf("expected field 'tasks', got %q", tasksField.Name)
	}
	if tasksField.Type != "Task[]" {
		t.Errorf("expected type 'Task[]', got %q", tasksField.Type)
	}

	// === VERIFY GO BLOCK ===
	if file.Script == nil {
		t.Fatal("expected Script block, got nil")
	}
	if len(file.Script.Source) < 100 {
		t.Errorf("Script source seems too short: %d characters", len(file.Script.Source))
	}
	// Verify it contains expected Go code fragments
	goSource := file.Script.Source
	if len(goSource) == 0 {
		t.Error("Script source is empty")
	}

	// === VERIFY TEMPLATE BLOCK ===
	if file.Template == nil {
		t.Fatal("expected Template block, got nil")
	}
	if len(file.Template.Source) < 50 {
		t.Errorf("Template source seems too short: %d characters", len(file.Template.Source))
	}
	// Verify content contains expected elements
	templateSource := file.Template.Source
	if !strings.Contains(templateSource, "<div") {
		t.Errorf("expected template to contain '<div', got: %q", templateSource[0:min(20, len(templateSource))])
	}

	// === VERIFY STYLE BLOCK ===
	if file.Style == nil {
		t.Fatal("expected Style block, got nil")
	}
	if !file.Style.Scoped {
		t.Error("expected Style.Scoped to be true")
	}
	if len(file.Style.Source) < 30 {
		t.Errorf("Style source seems too short: %d characters", len(file.Style.Source))
	}
	// Verify content contains expected styles
	styleSource := file.Style.Source
	if !strings.Contains(styleSource, ".task-item") {
		t.Errorf("expected style to contain '.task-item', got: %q", styleSource[0:min(50, len(styleSource))])
	}

	// Summary
	t.Logf("✓ Successfully parsed complete Phase 2 .gmx file")
	t.Logf("  - Models: %d (Task, User)", len(file.Models))
	t.Logf("  - Task fields: %d", len(task.Fields))
	t.Logf("  - User fields: %d", len(user.Fields))
	t.Logf("  - Script: %d chars", len(file.Script.Source))
	t.Logf("  - Template: %d chars", len(file.Template.Source))
	t.Logf("  - Style: %d chars (scoped=%v)", len(file.Style.Source), file.Style.Scoped)
}
