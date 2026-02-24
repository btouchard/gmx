package utils

import "strings"

// ToPascalCase converts snake_case and camelCase to PascalCase
// Gère les cas spéciaux: "id" → "ID", "user_id" → "UserID", "tenant_id" → "TenantID"
// C'est LA version unique, utilisée partout.
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Handle common abbreviations first (before splitting)
	switch s {
	case "id":
		return "ID"
	case "userId":
		return "UserID"
	case "tenantId":
		return "TenantID"
	}

	// Check if it contains underscores (snake_case)
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		for i, part := range parts {
			if part != "" {
				parts[i] = Capitalize(part)
			}
		}
		return strings.Join(parts, "")
	}

	// Simple camelCase → PascalCase: capitalize first letter
	return Capitalize(s)
}

// Capitalize met en majuscule la première lettre. "id" → "ID" (cas spécial).
func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	// Handle special cases like "id" -> "ID"
	if strings.ToLower(s) == "id" {
		return "ID"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ReceiverName retourne la première lettre en minuscule d'un nom de modèle.
func ReceiverName(modelName string) string {
	if modelName == "" {
		return ""
	}
	return strings.ToLower(modelName[:1])
}
