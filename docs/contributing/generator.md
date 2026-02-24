# Generator

Le generator orchestre la production du code Go final à partir de l'AST. Il est organisé en modules spécialisés.

## Structure

```
generator/
├── generator.go      # Orchestrateur principal
├── analysis.go       # Analyse de l'AST
├── gen_imports.go    # Détection et génération imports
├── gen_helpers.go    # Helpers (UUID, email, etc.)
├── gen_models.go     # Models GORM
├── gen_services.go   # Services config
├── gen_handlers.go   # HTTP handlers
├── gen_template.go   # Template setup
└── gen_main.go       # Fonction main()
```

## generator.go — Orchestrateur

```go
func (g *Generator) Generate(file *ast.GMXFile) (string, error) {
    var b strings.Builder

    // 1. Compute routes ONCE
    routes := g.genRouteRegistry(file.Template.Source)

    // 2. Package + imports
    b.WriteString("package main\n\n")
    b.WriteString(g.genImports(file))

    // 3. Helpers
    b.WriteString(g.genHelpers(file))

    // 4. Models
    b.WriteString(g.genModels(file.Models))

    // 5. Services
    b.WriteString(g.genServices(file.Services))

    // 6. Script (transpilation)
    if file.Script != nil && file.Script.Funcs != nil {
        result := script.Transpile(file.Script, modelNames)
        b.WriteString(result.GoCode)
        b.WriteString(g.genScriptHandlers(file.Script))
    }

    // 7. Template
    if file.Template != nil {
        b.WriteString(g.genTemplateInit(routes))
        b.WriteString(g.genTemplateConst(file))
    }

    // 8. Page Data
    b.WriteString(g.genPageData(file.Models))

    // 9. Database variable
    b.WriteString("var db *gorm.DB\n\n")

    // 10. Handlers
    b.WriteString(g.genHandlers(file, routes))

    // 11. Main
    b.WriteString(g.genMain(file))

    // 12. Format with gofmt
    formatted, err := format.Source([]byte(b.String()))
    return string(formatted), err
}
```

## gen_imports.go

Détecte automatiquement les imports nécessaires :

```go
func (g *Generator) genImports(file *ast.GMXFile) string {
    imports := []string{}

    if len(file.Models) > 0 {
        imports = append(imports, "gorm.io/gorm")
        imports = append(imports, "gorm.io/driver/sqlite")
    }

    if file.Template != nil {
        imports = append(imports, "html/template")
        imports = append(imports, "net/http")
    }

    if g.needsUUIDHelper(file) {
        imports = append(imports, "crypto/rand")
        imports = append(imports, "encoding/base64")
    }

    if g.needsEmailHelper(file) {
        imports = append(imports, "regexp")
    }

    // ...

    return formatImports(imports)
}
```

## gen_models.go

Génère les structs GORM avec validation.

### Model Struct

```go
func (g *Generator) genModels(models []*ast.ModelDecl) string {
    var b strings.Builder

    for _, model := range models {
        b.WriteString(fmt.Sprintf("type %s struct {\n", model.Name))

        for _, field := range model.Fields {
            fieldName := utils.ToPascalCase(field.Name)
            goType := g.mapType(field.Type)
            gormTags := g.genGormTags(field, model.Name)

            b.WriteString(fmt.Sprintf("\t%s %s `gorm:\"%s\" json:\"%s\"`\n",
                fieldName, goType, gormTags, field.Name))
        }

        b.WriteString("}\n\n")

        // Validation
        b.WriteString(g.genValidation(model))

        // BeforeCreate hook
        b.WriteString(g.genBeforeCreate(model))
    }

    return b.String()
}
```

### Validation Method

```go
func (g *Generator) genValidation(model *ast.ModelDecl) string {
    validations := []string{}

    for _, field := range model.Fields {
        for _, ann := range field.Annotations {
            switch ann.Name {
            case "min":
                validations = append(validations, g.genMinValidation(field, ann))
            case "max":
                validations = append(validations, g.genMaxValidation(field, ann))
            case "email":
                validations = append(validations, g.genEmailValidation(field))
            }
        }
    }

    if len(validations) == 0 {
        return ""
    }

    // Generate Validate() method
    return formatValidation(model, validations)
}
```

## gen_services.go

Génère la config et les implémentations pour chaque service.

### Pour sqlite/postgres

```go
dbCfg := initDatabase()
db, err := gorm.Open(sqlite.Open(dbCfg.Url), &gorm.Config{})
```

### Pour smtp

Génère une implémentation complète avec `net/smtp` :

```go
func (m *mailerImpl) Send(to string, subject string, body string) error {
    msg := []byte("To: " + to + "\\r\\n" +
        "Subject: " + subject + "\\r\\n" +
        "MIME-Version: 1.0\\r\\n" +
        "Content-Type: text/plain; charset=\\"utf-8\\"\\r\\n" +
        "\\r\\n" +
        body)

    var auth smtp.Auth
    if m.config.Pass != "" {
        auth = smtp.PlainAuth("", "", m.config.Pass, m.config.Host)
    }

    from := "noreply@localhost"
    addr := m.config.Host

    return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
```

### Pour http

Génère un client HTTP avec méthodes Get/Post :

```go
type GitHubClient struct {
    config *GitHubConfig
    http   *http.Client
}

func (c *GitHubClient) Get(path string) (*http.Response, error) {
    req, _ := http.NewRequest("GET", c.config.BaseUrl+path, nil)
    if c.config.ApiKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
    }
    return c.http.Do(req)
}
```

## gen_handlers.go

Génère les wrappers HTTP pour chaque fonction script.

```go
func (g *Generator) genScriptHandlers(script *ast.ScriptBlock) string {
    var b strings.Builder

    for _, fn := range script.Funcs {
        method := g.detectHTTPMethod(fn.Name)  // create→POST, toggle→PATCH, etc.

        b.WriteString(fmt.Sprintf("func handle%s(w http.ResponseWriter, r *http.Request) {\n",
            utils.ToPascalCase(fn.Name)))

        // 1. Method guard
        b.WriteString(fmt.Sprintf("\tif r.Method != %q {\n", method))
        b.WriteString("\t\thttp.Error(w, \"Method not allowed\", 405)\n")
        b.WriteString("\t\treturn\n\t}\n\n")

        // 2. CSRF validation (if POST/PATCH/DELETE)
        if method != "GET" {
            b.WriteString(g.genCSRFValidation())
        }

        // 3. Extract parameters
        b.WriteString(g.genParamExtraction(fn.Params))

        // 4. Call script function
        b.WriteString("\tctx := &GMXContext{DB: db, Writer: w, Request: r}\n")
        b.WriteString(fmt.Sprintf("\tif err := %s(ctx", fn.Name))
        for _, param := range fn.Params {
            b.WriteString(fmt.Sprintf(", %s", param.Name))
        }
        b.WriteString("); err != nil {\n")
        b.WriteString("\t\thttp.Error(w, err.Error(), 500)\n")
        b.WriteString("\t\treturn\n\t}\n")
        b.WriteString("}\n\n")
    }

    return b.String()
}
```

## gen_template.go

### Route Registry

Détecte automatiquement les routes depuis le template :

```go
func (g *Generator) genRouteRegistry(templateSource string) map[string]string {
    routes := make(map[string]string)

    re := regexp.MustCompile(`\{\{route\s+` + "`" + `([^` + "`" + `]+)` + "`" + `\}\}|` +
        `\{\{route\s+"([^"]+)"\}\}`)

    matches := re.FindAllStringSubmatch(templateSource, -1)
    for _, match := range matches {
        routeName := match[1]
        if routeName == "" {
            routeName = match[2]
        }
        routes[routeName] = "/" + routeName
    }

    return routes
}
```

### Template Init

```go
func (g *Generator) genTemplateInit(routes map[string]string) string {
    var b strings.Builder

    b.WriteString("var routes = map[string]string{\n")
    for name, path := range routes {
        b.WriteString(fmt.Sprintf("\t%q: %q,\n", name, path))
    }
    b.WriteString("}\n\n")

    b.WriteString("var funcMap = template.FuncMap{\n")
    b.WriteString("\t\"route\": func(name string) string {\n")
    b.WriteString("\t\tif path, ok := routes[name]; ok {\n")
    b.WriteString("\t\t\treturn path\n")
    b.WriteString("\t\t}\n")
    b.WriteString("\t\treturn \"#unknown-route\"\n")
    b.WriteString("\t},\n")
    b.WriteString("}\n\n")

    return b.String()
}
```

## gen_main.go

Génère la fonction main() complète :

```go
func (g *Generator) genMain(file *ast.GMXFile) string {
    var b strings.Builder

    b.WriteString("func main() {\n")

    // 1. Database init (if models exist)
    if len(file.Models) > 0 {
        b.WriteString(g.genDatabaseInit(file.Services))
        b.WriteString(g.genAutoMigrate(file.Models))
    }

    // 2. Route registration
    b.WriteString("\thttp.HandleFunc(\"/\", handleRoot)\n")
    for _, fn := range file.Script.Funcs {
        handlerName := "handle" + utils.ToPascalCase(fn.Name)
        b.WriteString(fmt.Sprintf("\thttp.HandleFunc(\"/%s\", %s)\n", fn.Name, handlerName))
    }

    // 3. Server start
    b.WriteString("\tlog.Println(\"Server starting on :8080\")\n")
    b.WriteString("\tlog.Fatal(http.ListenAndServe(\":8080\", nil))\n")
    b.WriteString("}\n")

    return b.String()
}
```

## Helpers

### UUID Generation

```go
func generateUUID() string {
    b := make([]byte, 16)
    rand.Read(b)
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
```

### Email Validation

```go
func isValidEmail(email string) bool {
    re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$`)
    return re.MatchString(email)
}
```

## Optimisations Identifiées

Voir `AUDIT_REPORT.md` pour les duplications :

1. `genRouteRegistry()` appelé 3 fois → **1 seul appel** (fixé dans le code actuel)
2. `needsXxxHelper()` répété 4 fois → à généraliser
3. Regex compilation à chaque appel → compiler une fois

## Prochaines Étapes

- **[Script Transpiler](script-transpiler.md)** — Transpilation GMX → Go
- **[Testing](testing.md)** — Tests du generator
