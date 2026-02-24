package errors

import (
	"strings"
	"testing"
)

func TestPositionString(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		expected string
	}{
		{
			"with file",
			Position{File: "test.gmx", Line: 10, Column: 5},
			"test.gmx:10:5",
		},
		{
			"without file",
			Position{Line: 10, Column: 5},
			"10:5",
		},
		{
			"line 1 column 1",
			Position{Line: 1, Column: 1},
			"1:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.String()
			if result != tt.expected {
				t.Errorf("Position.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCompileErrorError(t *testing.T) {
	err := &CompileError{
		Pos:     Position{File: "test.gmx", Line: 10, Column: 5},
		Message: "unexpected token",
		Phase:   "lexer",
	}

	result := err.Error()
	expected := "[lexer] test.gmx:10:5: unexpected token"

	if result != expected {
		t.Errorf("CompileError.Error() = %q, want %q", result, expected)
	}
}

func TestErrorListNew(t *testing.T) {
	el := NewErrorList()
	if el == nil {
		t.Fatal("NewErrorList() returned nil")
	}
	// Errors slice can be nil or empty
	if len(el.Errors) != 0 {
		t.Errorf("NewErrorList() Errors length = %d, want 0", len(el.Errors))
	}
}

func TestErrorListAdd(t *testing.T) {
	el := NewErrorList()

	pos := Position{Line: 5, Column: 10}
	el.Add(pos, "parser", "expected semicolon")

	if len(el.Errors) != 1 {
		t.Fatalf("After Add(), len(Errors) = %d, want 1", len(el.Errors))
	}

	err := el.Errors[0]
	if err.Pos != pos {
		t.Errorf("Error position = %v, want %v", err.Pos, pos)
	}
	if err.Phase != "parser" {
		t.Errorf("Error phase = %q, want %q", err.Phase, "parser")
	}
	if err.Message != "expected semicolon" {
		t.Errorf("Error message = %q, want %q", err.Message, "expected semicolon")
	}
}

func TestErrorListHasErrors(t *testing.T) {
	el := NewErrorList()

	if el.HasErrors() {
		t.Error("Empty ErrorList should not have errors")
	}

	el.Add(Position{Line: 1}, "test", "error 1")

	if !el.HasErrors() {
		t.Error("ErrorList with 1 error should return true for HasErrors()")
	}
}

func TestErrorListString(t *testing.T) {
	el := NewErrorList()
	el.Add(Position{Line: 1, Column: 5}, "lexer", "unexpected character")
	el.Add(Position{Line: 3, Column: 10}, "parser", "expected '}'")

	result := el.String()

	// Check that both errors are in the output
	if !strings.Contains(result, "[lexer] 1:5: unexpected character") {
		t.Errorf("String() missing first error, got: %s", result)
	}
	if !strings.Contains(result, "[parser] 3:10: expected '}'") {
		t.Errorf("String() missing second error, got: %s", result)
	}
}

func TestErrorListStringEmpty(t *testing.T) {
	el := NewErrorList()
	result := el.String()

	if result != "" {
		t.Errorf("Empty ErrorList.String() = %q, want %q", result, "")
	}
}
