package generator

import (
	"fmt"
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/utils"
	"strings"
)

// genHandlers generates HTTP handlers
func (g *Generator) genHandlers(file *ast.GMXFile, routes map[string]string) string {
	var b strings.Builder

	// Build a set of script function names for quick lookup
	scriptFuncs := g.scriptFuncNames(file)

	// Generate handleIndex
	b.WriteString("func handleIndex(w http.ResponseWriter, r *http.Request) {\n")

	// Get or create CSRF token
	b.WriteString("\t// Get or create CSRF token\n")
	b.WriteString("\tcsrfToken := \"\"\n")
	b.WriteString("\tif cookie, err := r.Cookie(\"_csrf\"); err == nil {\n")
	b.WriteString("\t\tcsrfToken = cookie.Value\n")
	b.WriteString("\t} else {\n")
	b.WriteString("\t\tcsrfToken = generateCSRFToken()\n")
	b.WriteString("\t\thttp.SetCookie(w, &http.Cookie{\n")
	b.WriteString("\t\t\tName:     \"_csrf\",\n")
	b.WriteString("\t\t\tValue:    csrfToken,\n")
	b.WriteString("\t\t\tPath:     \"/\",\n")
	b.WriteString("\t\t\tHttpOnly: false,\n")
	b.WriteString("\t\t\tSameSite: http.SameSiteStrictMode,\n")
	b.WriteString("\t\t\tSecure:   r.TLS != nil,\n")
	b.WriteString("\t\t})\n")
	b.WriteString("\t}\n\n")

	if len(file.Models) > 0 {
		b.WriteString("\tdata := PageData{\n")
		b.WriteString("\t\tCSRFToken: csrfToken,\n")

		// Fetch data for each model
		for _, model := range file.Models {
			b.WriteString(fmt.Sprintf("\t\t// Fetch %s from database\n", model.Name))
			// For now, just initialize empty slices
			// In a real implementation, we'd fetch from DB
			b.WriteString(fmt.Sprintf("\t\t%ss: []%s{},\n", model.Name, model.Name))
		}

		b.WriteString("\t}\n\n")
		b.WriteString("\t// Fetch data from database\n")
		for _, model := range file.Models {
			b.WriteString(fmt.Sprintf("\tdb.Find(&data.%ss)\n", model.Name))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("\tdata := PageData{\n")
		b.WriteString("\t\tCSRFToken: csrfToken,\n")
		b.WriteString("\t}\n\n")
	}

	b.WriteString("\tw.Header().Set(\"Content-Type\", \"text/html; charset=utf-8\")\n")
	b.WriteString("\tif err := tmpl.Execute(w, data); err != nil {\n")
	b.WriteString("\t\tlog.Printf(\"template error: %v\", err)\n")
	b.WriteString("\t\thttp.Error(w, \"Internal Server Error\", http.StatusInternalServerError)\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	// Generate stub handlers for each route ONLY if there's NO matching script function
	for routeName := range routes {
		// Skip if there's a script function with the same name
		if scriptFuncs[routeName] {
			continue
		}

		handlerName := "handle" + utils.Capitalize(routeName)
		b.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", handlerName))

		// Add basic stub implementation for createPost as an example
		if routeName == "createPost" && len(file.Models) > 0 {
			// Find the Post model
			for _, model := range file.Models {
				if model.Name == "Post" {
					b.WriteString("\ttitle := r.FormValue(\"title\")\n")
					b.WriteString(fmt.Sprintf("\tpost := %s{\n", model.Name))
					b.WriteString("\t\tTitle: title,\n")
					b.WriteString("\t}\n\n")

					// Check if model has validation method
					hasValidation := g.hasValidation(model)
					if hasValidation {
						b.WriteString("\t// Validate input\n")
						b.WriteString("\tif err := post.Validate(); err != nil {\n")
						b.WriteString("\t\thttp.Error(w, err.Error(), http.StatusBadRequest)\n")
						b.WriteString("\t\treturn\n")
						b.WriteString("\t}\n\n")
					}

					b.WriteString("\tdb.Create(&post)\n\n")
					b.WriteString("\t// Return HTML fragment for HTMX swap\n")
					b.WriteString("\tfragment := fmt.Sprintf(`<div class=\"card\">%s</div>`, template.HTMLEscapeString(post.Title))\n")
					b.WriteString("\tw.Header().Set(\"Content-Type\", \"text/html; charset=utf-8\")\n")
					b.WriteString("\tfmt.Fprint(w, fragment)\n")
					break
				}
			}
		} else {
			b.WriteString("\tw.WriteHeader(http.StatusOK)\n")
			b.WriteString(fmt.Sprintf("\tfmt.Fprintf(w, \"Handler for %s\")\n", routeName))
		}

		b.WriteString("}\n\n")
	}

	return b.String()
}

// genScriptHandlers generates HTTP handler wrappers for transpiled script functions
func (g *Generator) genScriptHandlers(scriptBlock *ast.ScriptBlock) string {
	var b strings.Builder

	for _, fn := range scriptBlock.Funcs {
		handlerName := "handle" + utils.Capitalize(fn.Name)
		expectedMethod := inferHTTPMethod(fn.Name)

		b.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", handlerName))

		// HTTP method guard
		b.WriteString(fmt.Sprintf("\t// Method guard\n"))
		b.WriteString(fmt.Sprintf("\tif r.Method != http.Method%s {\n", expectedMethod))
		b.WriteString("\t\thttp.Error(w, \"Method Not Allowed\", http.StatusMethodNotAllowed)\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n\n")

		b.WriteString("\tctx := &GMXContext{\n")
		b.WriteString("\t\tDB:      db,\n")
		b.WriteString("\t\tWriter:  w,\n")
		b.WriteString("\t\tRequest: r,\n")
		b.WriteString("\t}\n\n")

		// Extract parameters from request
		for _, param := range fn.Params {
			b.WriteString(fmt.Sprintf("\t// Extract parameter: %s\n", param.Name))
			b.WriteString(fmt.Sprintf("\t%s := r.PathValue(%q)\n", param.Name, param.Name))
			b.WriteString(fmt.Sprintf("\tif %s == \"\" {\n", param.Name))
			b.WriteString(fmt.Sprintf("\t\t%s = r.FormValue(%q)\n", param.Name, param.Name))
			b.WriteString("\t}\n")

			// Validate non-empty
			b.WriteString(fmt.Sprintf("\tif %s == \"\" {\n", param.Name))
			b.WriteString(fmt.Sprintf("\t\thttp.Error(w, \"Missing required parameter: %s\", http.StatusBadRequest)\n", param.Name))
			b.WriteString("\t\treturn\n")
			b.WriteString("\t}\n")

			// Handle type conversions and validations
			switch param.Type {
			case "uuid":
				b.WriteString(fmt.Sprintf("\tif !isValidUUID(%s) {\n", param.Name))
				b.WriteString("\t\thttp.Error(w, \"Invalid ID format\", http.StatusBadRequest)\n")
				b.WriteString("\t\treturn\n")
				b.WriteString("\t}\n")
			case "int":
				b.WriteString(fmt.Sprintf("\t%sInt, err := strconv.Atoi(%s)\n", param.Name, param.Name))
				b.WriteString("\tif err != nil {\n")
				b.WriteString("\t\thttp.Error(w, \"Invalid integer parameter\", http.StatusBadRequest)\n")
				b.WriteString("\t\treturn\n")
				b.WriteString("\t}\n")
			case "bool":
				b.WriteString(fmt.Sprintf("\t%sBool, err := strconv.ParseBool(%s)\n", param.Name, param.Name))
				b.WriteString("\tif err != nil {\n")
				b.WriteString("\t\thttp.Error(w, \"Invalid boolean parameter\", http.StatusBadRequest)\n")
				b.WriteString("\t\treturn\n")
				b.WriteString("\t}\n")
			}
		}
		b.WriteString("\n")

		// Call the business logic function
		b.WriteString(fmt.Sprintf("\tif err := %s(ctx", fn.Name))
		for _, param := range fn.Params {
			if param.Type == "int" {
				b.WriteString(fmt.Sprintf(", %sInt", param.Name))
			} else if param.Type == "bool" {
				b.WriteString(fmt.Sprintf(", %sBool", param.Name))
			} else {
				b.WriteString(fmt.Sprintf(", %s", param.Name))
			}
		}
		b.WriteString("); err != nil {\n")
		b.WriteString("\t\tlog.Printf(\"handler error: %v\", err)\n")
		b.WriteString("\t\thttp.Error(w, \"Internal Server Error\", http.StatusInternalServerError)\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n\n")
	}

	return b.String()
}

// inferHTTPMethod infers the HTTP method from a function name
// Returns the method name suitable for http.Method* constants (e.g., "Post", "Get", "Delete")
func inferHTTPMethod(funcName string) string {
	lower := strings.ToLower(funcName)

	// Check prefixes for specific methods
	if strings.HasPrefix(lower, "create") || strings.HasPrefix(lower, "add") {
		return "Post"
	}
	if strings.HasPrefix(lower, "toggle") || strings.HasPrefix(lower, "update") || strings.HasPrefix(lower, "edit") {
		return "Patch"
	}
	if strings.HasPrefix(lower, "delete") || strings.HasPrefix(lower, "remove") {
		return "Delete"
	}
	if strings.HasPrefix(lower, "list") || strings.HasPrefix(lower, "get") || strings.HasPrefix(lower, "find") {
		return "Get"
	}

	// Default to POST for mutations (safer)
	return "Post"
}
