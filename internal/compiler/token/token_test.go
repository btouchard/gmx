package token

import "testing"

func TestLookupIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		// Keywords
		{"func", FUNC},
		{"let", LET},
		{"const", CONST},
		{"true", TRUE},
		{"false", FALSE},
		{"if", IF},
		{"else", ELSE},
		{"return", RETURN},
		{"model", MODEL},
		{"service", SERVICE},
		{"import", IMPORT},
		{"task", TASK},
		{"as", AS},
		{"try", TRY},
		{"render", RENDER},
		{"ctx", CTX},
		{"error", ERROR},
		// Non-keywords
		{"variable", IDENT},
		{"Task", IDENT},
		{"userId", IDENT},
		{"foo_bar", IDENT},
		{"", IDENT},
		{"unknown", IDENT},
	}

	for _, tt := range tests {
		result := LookupIdent(tt.input)
		if result != tt.expected {
			t.Errorf("LookupIdent(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
