package lexer

import (
	"gmx/internal/compiler/token"
	"testing"
)

// TestCompleteWorkflow demonstrates the lexer handling all GMX sections correctly
func TestCompleteWorkflow(t *testing.T) {
	input := `model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3) @max(255)
  done:  bool   @default(false)
}

<script>
// Pure Go code block
func toggleTask(w http.ResponseWriter, r *http.Request) error {
    return nil
}
</script>

<template>
  <div class="task-item">{{.Title}}</div>
</template>

<style scoped>
  .task-item { padding: 1rem; }
</style>`

	l := New(input)

	// Verify we get MODEL section tokens
	tok := l.NextToken()
	if tok.Type != token.MODEL {
		t.Fatalf("Expected MODEL, got %s", tok.Type)
	}

	// Skip to first raw section (script)
	for tok.Type != token.RAW_GO && tok.Type != token.EOF {
		tok = l.NextToken()
	}
	if tok.Type != token.RAW_GO {
		t.Fatal("Never found RAW_GO")
	}

	// Should get RAW_TEMPLATE
	tok = l.NextToken()
	if tok.Type != token.RAW_TEMPLATE {
		t.Fatalf("Expected RAW_TEMPLATE, got %s", tok.Type)
	}

	// Should get RAW_STYLE
	tok = l.NextToken()
	if tok.Type != token.RAW_STYLE {
		t.Fatalf("Expected RAW_STYLE, got %s", tok.Type)
	}

	// Should reach EOF
	tok = l.NextToken()
	if tok.Type != token.EOF {
		t.Fatalf("Expected EOF, got %s", tok.Type)
	}

	t.Log("✓ Complete GMX file workflow verified")
}
