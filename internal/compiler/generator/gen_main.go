package generator

import (
	"fmt"
	"github.com/btouchard/gmx/internal/compiler/ast"
	"github.com/btouchard/gmx/internal/compiler/utils"
	"strings"
)

// genMain generates the main function
func (g *Generator) genMain(file *ast.GMXFile, routes map[string]string) string {
	var b strings.Builder

	b.WriteString("func main() {\n")

	// Find Database service if it exists
	dbService := g.findDatabaseService(file.Services)

	// Initialize services
	if len(file.Services) > 0 {
		for _, svc := range file.Services {
			varName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Cfg"
			b.WriteString(fmt.Sprintf("\t%s := init%s()\n", varName, svc.Name))

			// If service has methods, create the service instance
			if len(svc.Methods) > 0 {
				if svc.Provider == "smtp" {
					svcVarName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Svc"
					b.WriteString(fmt.Sprintf("\t%s := new%sService(%s)\n", svcVarName, svc.Name, varName))
				} else if svc.Provider == "http" {
					// HTTP clients use different factory name
					clientVarName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Client"
					b.WriteString(fmt.Sprintf("\t%s := new%sClient(%s)\n", clientVarName, svc.Name, varName))
				} else {
					// Generic services
					svcVarName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Svc"
					b.WriteString(fmt.Sprintf("\t%s := new%sService(%s)\n", svcVarName, svc.Name, varName))
				}
			} else if svc.Provider == "http" {
				// HTTP client without methods
				clientVarName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Client"
				b.WriteString(fmt.Sprintf("\t%s := new%sClient(%s)\n", clientVarName, svc.Name, varName))
			}
		}
		b.WriteString("\n")

		// Suppress unused variable warnings
		for _, svc := range file.Services {
			// Skip Database service config vars only if they're actually used (when models exist)
			if (svc.Provider == "postgres" || svc.Provider == "sqlite" || svc.Provider == "mysql") && len(file.Models) > 0 {
				continue
			}

			varName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Cfg"
			b.WriteString(fmt.Sprintf("\t_ = %s\n", varName))

			if len(svc.Methods) > 0 || svc.Provider == "http" {
				if svc.Provider == "http" {
					clientVarName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Client"
					b.WriteString(fmt.Sprintf("\t_ = %s\n", clientVarName))
				} else {
					svcVarName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Svc"
					b.WriteString(fmt.Sprintf("\t_ = %s\n", svcVarName))
				}
			}
		}
		b.WriteString("\n")
	}

	// Database setup if models exist
	if len(file.Models) > 0 {
		b.WriteString("\tvar err error\n")

		if dbService != nil {
			// Use Database service configuration
			dbVarName := strings.ToLower(dbService.Name[:1]) + dbService.Name[1:] + "Cfg"

			// Determine the driver based on provider
			switch dbService.Provider {
			case "postgres":
				b.WriteString(fmt.Sprintf("\tdb, err = gorm.Open(postgres.Open(%s.Url), &gorm.Config{})\n", dbVarName))
			case "mysql":
				b.WriteString(fmt.Sprintf("\tdb, err = gorm.Open(mysql.Open(%s.Url), &gorm.Config{})\n", dbVarName))
			default: // sqlite
				b.WriteString(fmt.Sprintf("\tdb, err = gorm.Open(sqlite.Open(%s.Url), &gorm.Config{})\n", dbVarName))
			}
		} else {
			// Fallback to hardcoded SQLite for backward compatibility
			b.WriteString("\tdb, err = gorm.Open(sqlite.Open(\"gmx.db\"), &gorm.Config{})\n")
		}

		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\tlog.Fatal(\"failed to connect database:\", err)\n")
		b.WriteString("\t}\n\n")

		// AutoMigrate all models
		b.WriteString("\tdb.AutoMigrate(")
		for i, model := range file.Models {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("&%s{}", model.Name))
		}
		b.WriteString(")\n\n")
	}

	// Build a map of all routes to register (avoid duplicates)
	routesToRegister := make(map[string]string) // path -> handlerName

	// Create mux
	b.WriteString("\tmux := http.NewServeMux()\n")

	// Register index route
	b.WriteString("\tmux.HandleFunc(\"/\", handleIndex)\n")

	// Register template routes
	// If a script function exists for the route, use the script handler wrapper
	// Otherwise use the stub handler
	for routeName, path := range routes {
		handlerName := "handle" + utils.Capitalize(routeName)
		routesToRegister[path] = handlerName
	}

	// Register script function handlers that are NOT in template routes
	if file.Script != nil && file.Script.Funcs != nil {
		for _, fn := range file.Script.Funcs {
			// Skip utility functions (non-error return type)
			if fn.ReturnType != "" && fn.ReturnType != "error" {
				continue
			}
			// Check if this function is already registered via template route
			found := false
			for routeName := range routes {
				if routeName == fn.Name {
					found = true
					break
				}
			}
			// If not found in template routes, register a default route
			if !found {
				apiPath := "/api/" + fn.Name
				handlerName := "handle" + utils.Capitalize(fn.Name)
				routesToRegister[apiPath] = handlerName
			}
		}
	}

	// Output all route registrations
	for path, handlerName := range routesToRegister {
		b.WriteString(fmt.Sprintf("\tmux.HandleFunc(%q, %s)\n", path, handlerName))
	}

	b.WriteString("\n")
	b.WriteString("\tfmt.Println(\"GMX server starting on :8080\")\n")
	b.WriteString("\tlog.Fatal(http.ListenAndServe(\":8080\", csrfProtect(securityHeaders(mux))))\n")
	b.WriteString("}\n")

	return b.String()
}
