package generator

import (
	"fmt"
	"github.com/btouchard/gmx/internal/compiler/ast"
	"strings"
)

// genImports generates the import block
func (g *Generator) genImports(file *ast.GMXFile) string {
	var b strings.Builder

	// First, generate GMX import comments (before the Go import block)
	if len(file.Imports) > 0 {
		b.WriteString("// ========== GMX Imports ==========\n")
		for _, imp := range file.Imports {
			if imp.IsNative {
				// Native Go imports will be added to the import block below
				b.WriteString(fmt.Sprintf("// Native Go import: %s as %s\n", imp.Path, imp.Alias))
			} else if imp.Default != "" {
				// Component import (Vue-style default import)
				b.WriteString(fmt.Sprintf("// TODO: Component import: %s from %s\n", imp.Default, imp.Path))
			} else if len(imp.Members) > 0 {
				// Destructured import
				membersStr := strings.Join(imp.Members, ", ")
				b.WriteString(fmt.Sprintf("// TODO: Destructured import: %s from %s\n", membersStr, imp.Path))
			}
		}
		b.WriteString("\n")
	}

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

	// Add native Go imports from GMX import declarations
	for _, imp := range file.Imports {
		if imp.IsNative {
			// Generate: import Alias "package/path"
			b.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
		}
	}

	b.WriteString(")\n")
	return b.String()
}
