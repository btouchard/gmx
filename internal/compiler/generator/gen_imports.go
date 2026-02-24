package generator

import (
	"gmx/internal/compiler/ast"
	"strings"
)

// genImports generates the import block
func (g *Generator) genImports(file *ast.GMXFile) string {
	var b strings.Builder

	b.WriteString("import (\n")

	// Always include crypto/rand for CSRF token generation (and UUID if needed)
	b.WriteString("\t\"crypto/rand\"\n")
	b.WriteString("\t\"fmt\"\n")

	// Add io for HTTP client
	if g.hasServiceWithProvider(file, "http") {
		b.WriteString("\t\"io\"\n")
	}

	b.WriteString("\t\"log\"\n")
	b.WriteString("\t\"net/http\"\n")

	// Add net/smtp for SMTP service
	if g.hasServiceWithProvider(file, "smtp") {
		b.WriteString("\t\"net/smtp\"\n")
	}

	// Add os import if services use @env
	if g.hasServicesWithEnv(file) {
		b.WriteString("\t\"os\"\n")
	}

	// Conditionally add regexp for email validation
	needsEmail := g.hasAnnotationMatch(file, func(a *ast.Annotation) bool {
		return a.Name == "email"
	})
	if needsEmail {
		b.WriteString("\t\"regexp\"\n")
	}

	// Conditionally add strconv for script parameter parsing
	if g.needsStrconv(file) {
		b.WriteString("\t\"strconv\"\n")
	}

	if file.Template != nil {
		b.WriteString("\t\"html/template\"\n")
	}

	if len(file.Models) > 0 || g.hasServiceWithProvider(file, "http") {
		b.WriteString("\t\"time\"\n")
	}

	// Database imports
	if len(file.Models) > 0 {
		b.WriteString("\t\"gorm.io/gorm\"\n")

		// Determine which database driver to import
		dbService := g.findDatabaseService(file.Services)
		if dbService != nil {
			switch dbService.Provider {
			case "postgres":
				b.WriteString("\t\"gorm.io/driver/postgres\"\n")
			case "mysql":
				b.WriteString("\t\"gorm.io/driver/mysql\"\n")
			default: // sqlite
				b.WriteString("\t\"gorm.io/driver/sqlite\"\n")
			}
		} else {
			// Default to SQLite for backward compatibility
			b.WriteString("\t\"gorm.io/driver/sqlite\"\n")
		}
	}

	b.WriteString(")\n")
	return b.String()
}
