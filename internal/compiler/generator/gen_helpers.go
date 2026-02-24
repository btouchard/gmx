package generator

import (
	"github.com/btouchard/gmx/internal/compiler/ast"
	"strings"
)

// genHelpers generates shared helper functions needed by the models
func (g *Generator) genHelpers(file *ast.GMXFile) string {
	var b strings.Builder

	needsUUID := g.hasAnnotationMatch(file, func(a *ast.Annotation) bool {
		return a.Name == "default" && a.SimpleArg() == "uuid_v4"
	})
	needsEmail := g.hasAnnotationMatch(file, func(a *ast.Annotation) bool {
		return a.Name == "email"
	})
	needsScoped := g.hasAnnotationMatch(file, func(a *ast.Annotation) bool {
		return a.Name == "scoped"
	})
	needsUUIDValidation := g.needsUUIDValidation(file)

	// Always generate helpers section (at minimum for securityHeaders)
	b.WriteString("// ========== Helper Functions ==========\n\n")

	if needsUUID {
		b.WriteString("// generateUUID generates a UUID v4 string\n")
		b.WriteString("func generateUUID() string {\n")
		b.WriteString("\tb := make([]byte, 16)\n")
		b.WriteString("\trand.Read(b)\n")
		b.WriteString("\tb[6] = (b[6] & 0x0f) | 0x40\n")
		b.WriteString("\tb[8] = (b[8] & 0x3f) | 0x80\n")
		b.WriteString("\treturn fmt.Sprintf(\"%08x-%04x-%04x-%04x-%012x\", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])\n")
		b.WriteString("}\n\n")
	}

	if needsEmail {
		b.WriteString("// isValidEmail checks if a string looks like a valid email\n")
		b.WriteString("func isValidEmail(email string) bool {\n")
		b.WriteString("\tre := regexp.MustCompile(`^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$`)\n")
		b.WriteString("\treturn re.MatchString(email)\n")
		b.WriteString("}\n\n")
	}

	if needsScoped {
		b.WriteString("// scopedDB returns a DB handle filtered by tenant_id for multi-tenant isolation\n")
		b.WriteString("func scopedDB(db *gorm.DB, tenantID string) *gorm.DB {\n")
		b.WriteString("\treturn db.Where(\"tenant_id = ?\", tenantID)\n")
		b.WriteString("}\n\n")
	}

	// UUID validation helper (for security)
	if needsUUIDValidation {
		b.WriteString("// isValidUUID checks if a string is a valid UUID v4 format\n")
		b.WriteString("func isValidUUID(s string) bool {\n")
		b.WriteString("\tif len(s) != 36 {\n")
		b.WriteString("\t\treturn false\n")
		b.WriteString("\t}\n")
		b.WriteString("\tfor i, c := range s {\n")
		b.WriteString("\t\tif i == 8 || i == 13 || i == 18 || i == 23 {\n")
		b.WriteString("\t\t\tif c != '-' {\n")
		b.WriteString("\t\t\t\treturn false\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t} else {\n")
		b.WriteString("\t\t\tif !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {\n")
		b.WriteString("\t\t\t\treturn false\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn true\n")
		b.WriteString("}\n\n")
	}

	// CSRF token generation (always included for security)
	b.WriteString("// generateCSRFToken generates a cryptographically secure random token\n")
	b.WriteString("func generateCSRFToken() string {\n")
	b.WriteString("\tb := make([]byte, 32)\n")
	b.WriteString("\trand.Read(b)\n")
	b.WriteString("\treturn fmt.Sprintf(\"%x\", b)\n")
	b.WriteString("}\n\n")

	// CSRF protection middleware
	b.WriteString("// csrfProtect is a middleware that provides CSRF protection using double-submit cookie pattern\n")
	b.WriteString("func csrfProtect(next http.Handler) http.Handler {\n")
	b.WriteString("\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\t\t// Safe methods: pass through, ensure cookie exists\n")
	b.WriteString("\t\tif r.Method == \"GET\" || r.Method == \"HEAD\" || r.Method == \"OPTIONS\" {\n")
	b.WriteString("\t\t\t// Set CSRF cookie if not present\n")
	b.WriteString("\t\t\tif _, err := r.Cookie(\"_csrf\"); err != nil {\n")
	b.WriteString("\t\t\t\ttoken := generateCSRFToken()\n")
	b.WriteString("\t\t\t\thttp.SetCookie(w, &http.Cookie{\n")
	b.WriteString("\t\t\t\t\tName:     \"_csrf\",\n")
	b.WriteString("\t\t\t\t\tValue:    token,\n")
	b.WriteString("\t\t\t\t\tPath:     \"/\",\n")
	b.WriteString("\t\t\t\t\tHttpOnly: false, // JS needs to read it for HTMX header\n")
	b.WriteString("\t\t\t\t\tSameSite: http.SameSiteStrictMode,\n")
	b.WriteString("\t\t\t\t\tSecure:   r.TLS != nil,\n")
	b.WriteString("\t\t\t\t})\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tnext.ServeHTTP(w, r)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Mutating methods: validate CSRF token\n")
	b.WriteString("\t\tcookie, err := r.Cookie(\"_csrf\")\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\thttp.Error(w, \"Forbidden - missing CSRF cookie\", http.StatusForbidden)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Check header first (HTMX sends this), then form value (regular forms)\n")
	b.WriteString("\t\ttoken := r.Header.Get(\"X-CSRF-Token\")\n")
	b.WriteString("\t\tif token == \"\" {\n")
	b.WriteString("\t\t\ttoken = r.FormValue(\"_csrf\")\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tif token == \"\" || token != cookie.Value {\n")
	b.WriteString("\t\t\thttp.Error(w, \"Forbidden - invalid CSRF token\", http.StatusForbidden)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tnext.ServeHTTP(w, r)\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")

	// Security headers middleware
	b.WriteString("// securityHeaders adds security headers to HTTP responses\n")
	b.WriteString("func securityHeaders(next http.Handler) http.Handler {\n")
	b.WriteString("\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\t\tw.Header().Set(\"X-Content-Type-Options\", \"nosniff\")\n")
	b.WriteString("\t\tw.Header().Set(\"X-Frame-Options\", \"DENY\")\n")
	b.WriteString("\t\tw.Header().Set(\"X-XSS-Protection\", \"1; mode=block\")\n")
	b.WriteString("\t\tnext.ServeHTTP(w, r)\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")

	return b.String()
}
