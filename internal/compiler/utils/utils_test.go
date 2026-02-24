package utils

import "testing"

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Simple cases
		{"simple id", "id", "ID"},
		{"simple email", "email", "Email"},
		{"simple title", "title", "Title"},

		// snake_case
		{"user_id snake", "user_id", "UserID"},
		{"tenant_id snake", "tenant_id", "TenantID"},
		{"created_at snake", "created_at", "CreatedAt"},
		{"updated_at snake", "updated_at", "UpdatedAt"},
		{"is_active snake", "is_active", "IsActive"},
		{"first_name snake", "first_name", "FirstName"},

		// camelCase
		{"userId camel", "userId", "UserID"},
		{"tenantId camel", "tenantId", "TenantID"},
		{"createdAt camel", "createdAt", "CreatedAt"},
		{"firstName camel", "firstName", "FirstName"},
		{"isActive camel", "isActive", "IsActive"},

		// edge cases
		{"empty string", "", ""},
		{"single char", "a", "A"},
		{"already Pascal", "UserID", "UserID"},

		// complex cases
		{"multiple underscores", "some_field_name", "SomeFieldName"},
		{"trailing underscore", "field_", "Field"},
		{"leading underscore", "_field", "Field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple word", "hello", "Hello"},
		{"id special", "id", "ID"},
		{"ID already caps", "ID", "ID"},
		{"Id mixed", "Id", "ID"},
		{"empty string", "", ""},
		{"single char", "a", "A"},
		{"already capitalized", "Hello", "Hello"},
		{"email", "email", "Email"},
		{"title", "title", "Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Capitalize(tt.input)
			if result != tt.expected {
				t.Errorf("Capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReceiverName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Task", "Task", "t"},
		{"User", "User", "u"},
		{"Post", "Post", "p"},
		{"Comment", "Comment", "c"},
		{"empty string", "", ""},
		{"single char", "A", "a"},
		{"lowercase already", "task", "t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReceiverName(tt.input)
			if result != tt.expected {
				t.Errorf("ReceiverName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
