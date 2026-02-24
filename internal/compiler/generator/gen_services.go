package generator

import (
	"fmt"
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/utils"
	"strings"
)

// genServices generates service config structs, init functions, and interfaces
func (g *Generator) genServices(services []*ast.ServiceDecl) string {
	var b strings.Builder

	for i, svc := range services {
		if i > 0 {
			b.WriteString("\n")
		}

		// Always generate config + init
		b.WriteString(g.genServiceConfig(svc))
		b.WriteString("\n")
		b.WriteString(g.genServiceInit(svc))
		b.WriteString("\n")

		// Provider-specific generation
		switch svc.Provider {
		case "smtp":
			if len(svc.Methods) > 0 {
				b.WriteString(g.genServiceInterface(svc))
				b.WriteString("\n")
				b.WriteString(g.genSMTPImpl(svc))
				b.WriteString("\n")
			}
		case "http":
			b.WriteString(g.genHTTPClient(svc))
			b.WriteString("\n")
		case "postgres", "sqlite", "mysql":
			// Database — no interface/stub needed, handled in genMain
		default:
			// Unknown provider — generate interface + stub
			if len(svc.Methods) > 0 {
				b.WriteString(g.genServiceInterface(svc))
				b.WriteString("\n")
				b.WriteString(g.genServiceStub(svc))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// genServiceConfig generates the config struct for a service
func (g *Generator) genServiceConfig(svc *ast.ServiceDecl) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("// %sConfig holds configuration for the %s service\n", svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("type %sConfig struct {\n", svc.Name))
	b.WriteString("\tProvider string\n")

	for _, field := range svc.Fields {
		fieldName := utils.ToPascalCase(field.Name)
		goType := g.mapType(field.Type)
		b.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
	}

	b.WriteString("}\n")
	return b.String()
}

// genServiceInit generates the init function for a service
func (g *Generator) genServiceInit(svc *ast.ServiceDecl) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("// init%s initializes the %s service configuration\n", svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("func init%s() *%sConfig {\n", svc.Name, svc.Name))
	b.WriteString("\tcfg := &" + svc.Name + "Config{\n")
	b.WriteString(fmt.Sprintf("\t\tProvider: %q,\n", svc.Provider))
	b.WriteString("\t}\n")

	// Load env vars
	for _, field := range svc.Fields {
		if field.EnvVar != "" {
			fieldName := utils.ToPascalCase(field.Name)
			b.WriteString(fmt.Sprintf("\tcfg.%s = os.Getenv(%q)\n", fieldName, field.EnvVar))
			b.WriteString(fmt.Sprintf("\tif cfg.%s == \"\" {\n", fieldName))
			b.WriteString(fmt.Sprintf("\t\tlog.Fatal(\"missing required env var: %s\")\n", field.EnvVar))
			b.WriteString("\t}\n")
		}
	}

	b.WriteString("\treturn cfg\n")
	b.WriteString("}\n")
	return b.String()
}

// genServiceInterface generates the interface for a service with methods
func (g *Generator) genServiceInterface(svc *ast.ServiceDecl) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("// %sService defines the interface for the %s service\n", svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("type %sService interface {\n", svc.Name))

	for _, method := range svc.Methods {
		methodName := utils.ToPascalCase(method.Name)
		b.WriteString(fmt.Sprintf("\t%s(", methodName))

		// Parameters
		for i, param := range method.Params {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s %s", param.Name, g.mapType(param.Type)))
		}

		b.WriteString(")")

		// Return type
		if method.ReturnType != "" {
			b.WriteString(" " + g.mapType(method.ReturnType))
		}

		b.WriteString("\n")
	}

	b.WriteString("}\n")
	return b.String()
}

// genServiceStub generates a stub implementation for a service
func (g *Generator) genServiceStub(svc *ast.ServiceDecl) string {
	var b strings.Builder

	stubName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Stub"

	b.WriteString(fmt.Sprintf("// %s is a stub implementation of %sService\n", stubName, svc.Name))
	b.WriteString(fmt.Sprintf("type %s struct {\n", stubName))
	b.WriteString(fmt.Sprintf("\tconfig *%sConfig\n", svc.Name))
	b.WriteString("}\n\n")

	// Generate stub methods
	for _, method := range svc.Methods {
		methodName := utils.ToPascalCase(method.Name)
		b.WriteString(fmt.Sprintf("func (s *%s) %s(", stubName, methodName))

		// Parameters
		for i, param := range method.Params {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s %s", param.Name, g.mapType(param.Type)))
		}

		b.WriteString(")")

		// Return type
		if method.ReturnType != "" {
			b.WriteString(" " + g.mapType(method.ReturnType))
		}

		b.WriteString(" {\n")
		b.WriteString(fmt.Sprintf("\tlog.Printf(\"[%%s] %s.%s called (stub)\", s.config.Provider)\n", svc.Name, methodName))

		// Return appropriate zero value
		if method.ReturnType != "" {
			if method.ReturnType == "error" {
				b.WriteString("\treturn nil\n")
			} else {
				b.WriteString(fmt.Sprintf("\treturn %s\n", g.zeroValue(method.ReturnType)))
			}
		}

		b.WriteString("}\n\n")
	}

	// Generate factory function
	b.WriteString(fmt.Sprintf("// new%sService creates a new instance of %sService\n", svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("func new%sService(cfg *%sConfig) %sService {\n", svc.Name, svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("\treturn &%s{config: cfg}\n", stubName))
	b.WriteString("}\n")

	return b.String()
}

// zeroValue returns the zero value for a given type
func (g *Generator) zeroValue(t string) string {
	switch t {
	case "string":
		return "\"\""
	case "int":
		return "0"
	case "bool":
		return "false"
	case "error":
		return "nil"
	default:
		return "nil"
	}
}

// fieldExists checks if a service has a field with the given name
func fieldExists(svc *ast.ServiceDecl, name string) bool {
	for _, field := range svc.Fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

// genSMTPImpl generates the SMTP implementation for a mailer service
func (g *Generator) genSMTPImpl(svc *ast.ServiceDecl) string {
	var b strings.Builder

	implName := strings.ToLower(svc.Name[:1]) + svc.Name[1:] + "Impl"

	b.WriteString(fmt.Sprintf("// %s is an SMTP implementation of %sService\n", implName, svc.Name))
	b.WriteString(fmt.Sprintf("type %s struct {\n", implName))
	b.WriteString(fmt.Sprintf("\tconfig *%sConfig\n", svc.Name))
	b.WriteString("}\n\n")

	// Generate Send method (assuming it's the standard mailer signature)
	for _, method := range svc.Methods {
		if method.Name == "send" {
			methodName := utils.ToPascalCase(method.Name)
			b.WriteString(fmt.Sprintf("func (m *%s) %s(", implName, methodName))

			// Parameters
			for i, param := range method.Params {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(fmt.Sprintf("%s %s", param.Name, g.mapType(param.Type)))
			}

			b.WriteString(") error {\n")

			// SMTP implementation
			b.WriteString("\tmsg := []byte(\"To: \" + to + \"\\r\\n\" +\n")
			b.WriteString("\t\t\"Subject: \" + subject + \"\\r\\n\" +\n")
			b.WriteString("\t\t\"MIME-Version: 1.0\\r\\n\" +\n")
			b.WriteString("\t\t\"Content-Type: text/plain; charset=\\\"utf-8\\\"\\r\\n\" +\n")
			b.WriteString("\t\t\"\\r\\n\" +\n")
			b.WriteString("\t\tbody)\n\n")

			b.WriteString("\tvar auth smtp.Auth\n")
			b.WriteString("\tif m.config.Pass != \"\" {\n")

			// Check if User field exists
			if fieldExists(svc, "user") {
				b.WriteString("\t\tauth = smtp.PlainAuth(\"\", m.config.User, m.config.Pass, m.config.Host)\n")
			} else {
				b.WriteString("\t\tauth = smtp.PlainAuth(\"\", \"\", m.config.Pass, m.config.Host)\n")
			}
			b.WriteString("\t}\n\n")

			// From address
			b.WriteString("\tfrom := \"noreply@localhost\"\n")
			if fieldExists(svc, "from") {
				b.WriteString("\tif m.config.From != \"\" {\n")
				b.WriteString("\t\tfrom = m.config.From\n")
				b.WriteString("\t}\n\n")
			}

			// Host and port
			b.WriteString("\taddr := m.config.Host\n")
			if fieldExists(svc, "port") {
				b.WriteString("\tif m.config.Port != \"\" {\n")
				b.WriteString("\t\taddr = m.config.Host + \":\" + m.config.Port\n")
				b.WriteString("\t}\n\n")
			}

			b.WriteString("\treturn smtp.SendMail(addr, auth, from, []string{to}, msg)\n")
			b.WriteString("}\n\n")
		}
	}

	// Generate factory function
	b.WriteString(fmt.Sprintf("// new%sService creates a new SMTP instance of %sService\n", svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("func new%sService(cfg *%sConfig) %sService {\n", svc.Name, svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("\treturn &%s{config: cfg}\n", implName))
	b.WriteString("}\n")

	return b.String()
}

// genHTTPClient generates an HTTP client for external API services
func (g *Generator) genHTTPClient(svc *ast.ServiceDecl) string {
	var b strings.Builder

	clientName := svc.Name + "Client"

	b.WriteString(fmt.Sprintf("// %s is an HTTP client for the %s service\n", clientName, svc.Name))
	b.WriteString(fmt.Sprintf("type %s struct {\n", clientName))
	b.WriteString(fmt.Sprintf("\tconfig *%sConfig\n", svc.Name))
	b.WriteString("\thttp   *http.Client\n")
	b.WriteString("}\n\n")

	// Factory function
	b.WriteString(fmt.Sprintf("// new%sClient creates a new HTTP client for %s\n", svc.Name, svc.Name))
	b.WriteString(fmt.Sprintf("func new%sClient(cfg *%sConfig) *%s {\n", svc.Name, svc.Name, clientName))
	b.WriteString(fmt.Sprintf("\treturn &%s{\n", clientName))
	b.WriteString("\t\tconfig: cfg,\n")
	b.WriteString("\t\thttp:   &http.Client{Timeout: 30 * time.Second},\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	// GET method
	b.WriteString(fmt.Sprintf("// Get makes a GET request to the API\n"))
	b.WriteString(fmt.Sprintf("func (c *%s) Get(path string) (*http.Response, error) {\n", clientName))
	b.WriteString("\treq, err := http.NewRequest(\"GET\", c.config.BaseUrl+path, nil)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, err\n")
	b.WriteString("\t}\n")

	// Add API key if field exists
	if fieldExists(svc, "apiKey") {
		b.WriteString("\tif c.config.ApiKey != \"\" {\n")
		b.WriteString("\t\treq.Header.Set(\"Authorization\", \"Bearer \"+c.config.ApiKey)\n")
		b.WriteString("\t}\n")
	}

	b.WriteString("\treturn c.http.Do(req)\n")
	b.WriteString("}\n\n")

	// POST method
	b.WriteString(fmt.Sprintf("// Post makes a POST request to the API\n"))
	b.WriteString(fmt.Sprintf("func (c *%s) Post(path string, body io.Reader) (*http.Response, error) {\n", clientName))
	b.WriteString("\treq, err := http.NewRequest(\"POST\", c.config.BaseUrl+path, body)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treq.Header.Set(\"Content-Type\", \"application/json\")\n")

	// Add API key if field exists
	if fieldExists(svc, "apiKey") {
		b.WriteString("\tif c.config.ApiKey != \"\" {\n")
		b.WriteString("\t\treq.Header.Set(\"Authorization\", \"Bearer \"+c.config.ApiKey)\n")
		b.WriteString("\t}\n")
	}

	b.WriteString("\treturn c.http.Do(req)\n")
	b.WriteString("}\n")

	return b.String()
}
