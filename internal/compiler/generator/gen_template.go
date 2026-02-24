package generator

import (
	"fmt"
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/resolver"
	"regexp"
	"strings"
)

// routeRegex is compiled once at package level for efficiency
var routeRegex = regexp.MustCompile(`\{\{route\s+` + "`" + `([^` + "`" + `]+)` + "`" + `\}\}|\{\{route\s+"([^"]+)"\}\}`)

// genRouteRegistry scans template source for {{route `name`}} calls and extracts route names
func (g *Generator) genRouteRegistry(templateSource string) map[string]string {
	routes := make(map[string]string)

	// Use pre-compiled package-level regex
	matches := routeRegex.FindAllStringSubmatch(templateSource, -1)

	for _, match := range matches {
		var routeName string
		if match[1] != "" {
			routeName = match[1]
		} else if match[2] != "" {
			routeName = match[2]
		}
		if routeName != "" {
			routes[routeName] = "/api/" + routeName
		}
	}

	return routes
}

// genTemplateInit generates the template initialization code with FuncMap
func (g *Generator) genTemplateInit(routes map[string]string) string {
	var b strings.Builder

	b.WriteString("var tmpl *template.Template\n\n")
	b.WriteString("func init() {\n")
	b.WriteString("\tfuncMap := template.FuncMap{\n")
	b.WriteString("\t\t\"route\": func(name string) string {\n")
	b.WriteString("\t\t\troutes := map[string]string{\n")

	// Add all routes to the map
	for name, path := range routes {
		b.WriteString(fmt.Sprintf("\t\t\t\t%q: %q,\n", name, path))
	}

	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tif r, ok := routes[name]; ok {\n")
	b.WriteString("\t\t\t\treturn r\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn \"/api/\" + name\n")
	b.WriteString("\t\t},\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\ttmpl = template.Must(template.New(\"page\").Funcs(funcMap).Parse(pageTemplate))\n")
	b.WriteString("}\n")

	return b.String()
}

// genTemplateConst generates the pageTemplate constant with full HTML structure
func (g *Generator) genTemplateConst(file *ast.GMXFile, components map[string]*resolver.ComponentInfo) string {
	var b strings.Builder

	// Check if the template already contains a full HTML page
	templateSrc := ""
	if file.Template != nil {
		templateSrc = file.Template.Source
	}

	// Case-insensitive check for existing HTML structure
	lowerSrc := strings.ToLower(templateSrc)
	hasFullHTML := strings.Contains(lowerSrc, "<!doctype") || strings.Contains(lowerSrc, "<html")

	var htmlStr string

	if hasFullHTML {
		// Template already has full HTML - use it as-is, only inject CSS if needed
		// Merge component styles
		componentStyles := g.genComponentStyles(components)
		allStyles := ""
		if file.Style != nil && file.Style.Source != "" {
			allStyles = file.Style.Source + "\n" + componentStyles
		} else {
			allStyles = componentStyles
		}

		if allStyles != "" {
			// Find </head> and inject style before it
			headEndIdx := strings.Index(templateSrc, "</head>")
			if headEndIdx == -1 {
				headEndIdx = strings.Index(templateSrc, "</HEAD>")
			}

			if headEndIdx != -1 {
				// Inject style and CSRF protection before </head>
				var html strings.Builder
				html.WriteString(templateSrc[:headEndIdx])
				html.WriteString("  <style>\n")
				html.WriteString("  /* GMX Scoped Styles */\n")
				html.WriteString("  " + allStyles + "\n")
				html.WriteString("  </style>\n")
				// Inject CSRF protection
				html.WriteString("  <meta name=\"csrf-token\" content=\"{{.CSRFToken}}\">\n")
				html.WriteString("  <script>\n")
				html.WriteString("    document.addEventListener('DOMContentLoaded', function() {\n")
				html.WriteString("      document.body.addEventListener('htmx:configRequest', function(e) {\n")
				html.WriteString("        var token = document.querySelector('meta[name=\"csrf-token\"]');\n")
				html.WriteString("        if (token) {\n")
				html.WriteString("          e.detail.headers['X-CSRF-Token'] = token.content;\n")
				html.WriteString("        }\n")
				html.WriteString("      });\n")
				html.WriteString("    });\n")
				html.WriteString("  </script>\n")
				html.WriteString(templateSrc[headEndIdx:])
				htmlStr = html.String()
			} else {
				// No </head> found, just use template as-is
				htmlStr = templateSrc
			}
		} else {
			// No style to inject, but still inject CSRF protection
			headEndIdx := strings.Index(templateSrc, "</head>")
			if headEndIdx == -1 {
				headEndIdx = strings.Index(templateSrc, "</HEAD>")
			}

			if headEndIdx != -1 {
				// Inject CSRF protection before </head>
				var html strings.Builder
				html.WriteString(templateSrc[:headEndIdx])
				html.WriteString("  <meta name=\"csrf-token\" content=\"{{.CSRFToken}}\">\n")
				html.WriteString("  <script>\n")
				html.WriteString("    document.addEventListener('DOMContentLoaded', function() {\n")
				html.WriteString("      document.body.addEventListener('htmx:configRequest', function(e) {\n")
				html.WriteString("        var token = document.querySelector('meta[name=\"csrf-token\"]');\n")
				html.WriteString("        if (token) {\n")
				html.WriteString("          e.detail.headers['X-CSRF-Token'] = token.content;\n")
				html.WriteString("        }\n")
				html.WriteString("      });\n")
				html.WriteString("    });\n")
				html.WriteString("  </script>\n")
				html.WriteString(templateSrc[headEndIdx:])
				htmlStr = html.String()
			} else {
				// No </head> found, use template as-is
				htmlStr = templateSrc
			}
		}
	} else {
		// Template doesn't have full HTML - wrap it
		var html strings.Builder
		html.WriteString("<!DOCTYPE html>\n")
		html.WriteString("<html>\n")
		html.WriteString("<head>\n")
		html.WriteString("    <meta charset=\"UTF-8\">\n")
		html.WriteString("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
		html.WriteString("    <script src=\"https://cdn.tailwindcss.com\"></script>\n")
		html.WriteString("    <script src=\"https://unpkg.com/htmx.org@2.0.4\"></script>\n")

		// Merge component styles
		componentStyles := g.genComponentStyles(components)
		allStyles := ""
		if file.Style != nil && file.Style.Source != "" {
			allStyles = file.Style.Source + "\n" + componentStyles
		} else {
			allStyles = componentStyles
		}

		// Inject CSS if present
		if allStyles != "" {
			html.WriteString("    <style>\n")
			html.WriteString("    /* GMX Scoped Styles */\n")
			html.WriteString("    " + allStyles + "\n")
			html.WriteString("    </style>\n")
		}

		// Inject CSRF protection (always included)
		html.WriteString("    <meta name=\"csrf-token\" content=\"{{.CSRFToken}}\">\n")
		html.WriteString("    <script>\n")
		html.WriteString("      document.addEventListener('DOMContentLoaded', function() {\n")
		html.WriteString("        document.body.addEventListener('htmx:configRequest', function(e) {\n")
		html.WriteString("          var token = document.querySelector('meta[name=\"csrf-token\"]');\n")
		html.WriteString("          if (token) {\n")
		html.WriteString("            e.detail.headers['X-CSRF-Token'] = token.content;\n")
		html.WriteString("          }\n")
		html.WriteString("        });\n")
		html.WriteString("      });\n")
		html.WriteString("    </script>\n")

		html.WriteString("</head>\n")
		html.WriteString("<body class=\"p-4\">\n")
		html.WriteString("<!-- GMX Component Template -->\n")

		// Add template content
		html.WriteString(templateSrc)
		html.WriteString("\n")

		html.WriteString("</body>\n")
		html.WriteString("</html>")
		htmlStr = html.String()
	}

	// Append component template definitions
	if len(components) > 0 {
		htmlStr += "\n" + g.genComponentTemplates(components)
	}

	// Use const with string concatenation to handle backticks
	b.WriteString("const pageTemplate = ")
	b.WriteString(escapeTemplateString(htmlStr))
	b.WriteString("\n")

	return b.String()
}

// escapeTemplateString creates a Go string literal, handling backticks properly
func escapeTemplateString(s string) string {
	// If no backticks, use a simple raw string
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}

	// Otherwise, split around backticks and concatenate
	parts := strings.Split(s, "`")
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			// Add the backtick as a quoted string
			b.WriteString(" + \"`\" + ")
		}
		// Add the part as a raw string
		b.WriteString("`" + part + "`")
	}
	return b.String()
}

// genComponentTemplates generates {{define}} blocks for each component
func (g *Generator) genComponentTemplates(components map[string]*resolver.ComponentInfo) string {
	if len(components) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n<!-- ========== Component Templates ========== -->\n\n")

	for name, info := range components {
		if info.File.Template == nil {
			continue
		}

		b.WriteString(fmt.Sprintf("<!-- Component: %s (from %s) -->\n", name, info.Path))
		b.WriteString(fmt.Sprintf("{{define %q}}\n", name))
		b.WriteString(info.File.Template.Source)
		b.WriteString("\n{{end}}\n\n")
	}

	return b.String()
}

// genComponentStyles merges all component styles
func (g *Generator) genComponentStyles(components map[string]*resolver.ComponentInfo) string {
	if len(components) == 0 {
		return ""
	}

	var b strings.Builder

	for name, info := range components {
		if info.File.Style == nil || info.File.Style.Source == "" {
			continue
		}

		b.WriteString(fmt.Sprintf("\n/* Component: %s */\n", name))
		b.WriteString(info.File.Style.Source)
		b.WriteString("\n")
	}

	return b.String()
}
