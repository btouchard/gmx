# Architecture

GMX est un compilateur multi-phase qui transforme des fichiers `.gmx` en code Go compilable. Voici l'architecture complète du pipeline de compilation.

## Vue d'Ensemble

```
┌──────────┐      ┌───────┐      ┌────────┐      ┌───────────┐      ┌─────────┐
│ .gmx     │─────▶│ Lexer │─────▶│ Parser │─────▶│ Generator │─────▶│ Go Code │
│ Source   │      │       │      │        │      │           │      │         │
└──────────┘      └───────┘      └────────┘      └───────────┘      └─────────┘
                      │              │                 │
                      ▼              ▼                 ▼
                  Tokens           AST            Transpiler
                                    │                 │
                                    │                 ▼
                                    │          Script → Go
                                    │
                                    ▼
                            script.Parse()
```

## Pipeline de Compilation

### Phase 1: Lexing

**Fichier** : `internal/compiler/lexer/lexer.go`

**Responsabilité** : Transformer le texte brut en tokens

```go
input := `model Task { id: uuid @pk }`
lexer := lexer.New(input)

tokens := []
// TOKEN_MODEL "model"
// TOKEN_IDENT "Task"
// TOKEN_LBRACE "{"
// TOKEN_IDENT "id"
// TOKEN_COLON ":"
// TOKEN_IDENT "uuid"
// TOKEN_AT "@"
// TOKEN_IDENT "pk"
// TOKEN_RBRACE "}"
```

**Tokens Spéciaux** :
- `RAW_GO` — contenu de `<script>...</script>`
- `RAW_TEMPLATE` — contenu de `<template>...</template>`
- `RAW_STYLE` — contenu de `<style>...</style>`

Ces tokens contiennent **le contenu brut** sans parsing, pour déléguer aux phases suivantes.

### Phase 2: Parsing

**Fichier** : `internal/compiler/parser/parser.go`

**Responsabilité** : Construire l'AST depuis les tokens

```go
parser := parser.New(lexer)
file := parser.ParseGMXFile()

// file.Models = []*ast.ModelDecl
// file.Services = []*ast.ServiceDecl
// file.Script = *ast.ScriptBlock (avec Source brut)
// file.Template = *ast.TemplateBlock
// file.Style = *ast.StyleBlock
```

**Structures Parsées** :
- Models et fields
- Services et config
- Annotations (`@pk`, `@min(3)`, `@env("VAR")`)
- **Bloc Script** (mais pas encore le contenu GMX Script)

### Phase 2.5: Script Parsing

**Fichier** : `internal/compiler/script/parser.go`

**Responsabilité** : Parser le contenu GMX Script en AST de fonctions

Le parser principal appelle le script parser pour le contenu de `<script>` :

```go
// Dans parser/parser.go
case token.RAW_GO:
    source := p.curToken.Literal
    funcs, parseErrors := script.Parse(source, lineOffset)

    scriptBlock := &ast.ScriptBlock{
        Source:    source,
        Funcs:     funcs,
        StartLine: lineOffset,
    }
```

**Résultat** :
- `ScriptBlock.Source` — le code brut (fallback)
- `ScriptBlock.Funcs` — les fonctions parsées

### Phase 3: Transpilation

**Fichier** : `internal/compiler/script/transpiler.go`

**Responsabilité** : Convertir les fonctions GMX Script en fonctions Go

```go
// GMX Script
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}

// ↓↓↓ Transpilation ↓↓↓

// Go
func toggleTask(ctx *GMXContext, id string) error {
    task, err := TaskFind(ctx.DB, id)
    if err != nil {
        return err
    }
    task.Done = !task.Done
    if err := TaskSave(ctx.DB, task); err != nil {
        return err
    }
    return renderFragment(ctx.Writer, "task", task)
}
```

### Phase 4: Generation

**Fichier** : `internal/compiler/generator/generator.go` + modules

**Responsabilité** : Orchestrer la génération du fichier Go final

Le generator appelle différents sous-générateurs :

1. **Imports** (`gen_imports.go`) — détecte les dépendances
2. **Helpers** (`gen_helpers.go`) — UUID, email validation, etc.
3. **Models** (`gen_models.go`) — structs GORM + validation
4. **Services** (`gen_services.go`) — config + interfaces
5. **Script** (via transpiler) — fonctions métier
6. **Handlers** (`gen_handlers.go`) — HTTP wrappers
7. **Template** (`gen_template.go`) — template init + constantes
8. **Main** (`gen_main.go`) — fonction main() complète

**Code généré** :

```go
package main

import (...)

// ===== Helpers =====
func generateUUID() string { ... }
func isValidEmail(email string) bool { ... }

// ===== Models =====
type Task struct { ... }
func (t *Task) Validate() error { ... }
func (t *Task) BeforeCreate(tx *gorm.DB) error { ... }

// ===== Services =====
type DatabaseConfig struct { ... }
func initDatabase() *DatabaseConfig { ... }

// ===== Script (Transpiled) =====
type GMXContext struct { ... }
func TaskFind(db *gorm.DB, id string) (*Task, error) { ... }
func toggleTask(ctx *GMXContext, id string) error { ... }

// ===== Handlers =====
func handleToggleTask(w http.ResponseWriter, r *http.Request) { ... }

// ===== Template =====
var tmpl = template.Must(...)
const templateSource = `...`

// ===== Main =====
func main() {
    db, err := gorm.Open(...)
    db.AutoMigrate(&Task{})
    http.HandleFunc("/", handleRoot)
    http.HandleFunc("/toggleTask", handleToggleTask)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Structure des Packages

```
gmx/
├── cmd/
│   └── gmx/
│       └── main.go              # CLI entry point
├── internal/
│   └── compiler/
│       ├── token/
│       │   ├── token.go         # Token types et constantes
│       │   └── token_test.go
│       ├── lexer/
│       │   ├── lexer.go         # Lexer principal
│       │   └── lexer_test.go
│       ├── ast/
│       │   ├── ast.go           # Tous les types AST
│       │   └── ast_test.go
│       ├── parser/
│       │   ├── parser.go        # Parser GMX principal
│       │   └── parser_test.go
│       ├── script/
│       │   ├── parser.go        # Parser GMX Script
│       │   ├── transpiler.go    # Transpileur Script → Go
│       │   ├── parser_test.go
│       │   └── transpiler_test.go
│       ├── generator/
│       │   ├── generator.go     # Orchestrateur
│       │   ├── analysis.go      # Analyse de l'AST
│       │   ├── gen_imports.go   # Génération imports
│       │   ├── gen_helpers.go   # Helpers (UUID, email, etc.)
│       │   ├── gen_models.go    # Génération models GORM
│       │   ├── gen_services.go  # Génération services
│       │   ├── gen_handlers.go  # Génération HTTP handlers
│       │   ├── gen_template.go  # Génération template setup
│       │   ├── gen_main.go      # Génération main()
│       │   └── generator_test.go
│       ├── utils/
│       │   ├── utils.go         # PascalCase, ReceiverName, etc.
│       │   └── utils_test.go
│       └── errors/
│           ├── errors.go        # Error handling
│           └── errors_test.go
└── examples/
    ├── example.gmx
    ├── todo.gmx
    └── services.gmx
```

## Flow Détaillé

### Entrée : `cmd/gmx/main.go`

```go
func main() {
    inputFile := os.Args[1]
    data, _ := os.ReadFile(inputFile)

    // 1. Lexing
    l := lexer.New(string(data))

    // 2. Parsing
    p := parser.New(l)
    file := p.ParseGMXFile()

    // 3. Generation
    gen := generator.New()
    code, err := gen.Generate(file)

    // 4. Write output
    os.WriteFile("main.go", []byte(code), 0644)
}
```

### Génération : `generator/generator.go`

```go
func (g *Generator) Generate(file *ast.GMXFile) (string, error) {
    var b strings.Builder

    // 1. Package + imports
    b.WriteString("package main\n\n")
    b.WriteString(g.genImports(file))

    // 2. Helpers
    b.WriteString(g.genHelpers(file))

    // 3. Models
    b.WriteString(g.genModels(file.Models))

    // 4. Services
    b.WriteString(g.genServices(file.Services))

    // 5. Script (transpilation)
    if file.Script != nil && file.Script.Funcs != nil {
        result := script.Transpile(file.Script, modelNames)
        b.WriteString(result.GoCode)
        b.WriteString(g.genScriptHandlers(file.Script))
    }

    // 6. Template
    if file.Template != nil {
        b.WriteString(g.genTemplateInit(routes))
        b.WriteString(g.genTemplateConst(file))
    }

    // 7. Handlers
    b.WriteString(g.genHandlers(file, routes))

    // 8. Main
    b.WriteString(g.genMain(file))

    // 9. Format with gofmt
    formatted, err := format.Source([]byte(b.String()))
    return string(formatted), err
}
```

## Détails des Générateurs

### gen_imports.go

Détecte automatiquement les imports nécessaires :

```go
needsGorm := len(file.Models) > 0
needsHTTP := file.Template != nil
needsTime := g.hasDateTimeField(file.Models)
needsUUID := g.needsUUIDHelper(file)
needsEmail := g.needsEmailHelper(file)
needsSMTP := g.hasSMTPService(file.Services)
```

### gen_models.go

Génère :
- Struct GORM avec tags
- Méthode `Validate()` (si annotations présentes)
- Hook `BeforeCreate()` (si `@default(uuid_v4)`)

### gen_services.go

Génère pour chaque service :
- Struct de config
- Fonction `init<Service>()`
- Interface (si méthodes déclarées)
- Implémentation (SMTP, HTTP) ou stub

### gen_handlers.go

Génère pour chaque fonction script :

```go
func handle<FunctionName>(w http.ResponseWriter, r *http.Request) {
    // 1. Method guard
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", 405)
        return
    }

    // 2. CSRF validation
    // ...

    // 3. Extract parameters
    id := r.PathValue("id") || r.FormValue("id")

    // 4. Call script function
    ctx := &GMXContext{DB: db, Writer: w, Request: r}
    if err := functionName(ctx, id); err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
}
```

### gen_template.go

Génère :
- Route registry (détecté par regex)
- Template FuncMap avec helper `route`
- Parsing du template
- Injection du script CSRF HTMX

### gen_main.go

Génère :
- Init database
- AutoMigrate
- Route registration
- Server start

## Error Handling

### Parser Error Recovery

Le parser utilise `synchronize()` pour récupérer après une erreur :

```go
func (p *Parser) synchronize() {
    for !p.curTokenIs(token.EOF) {
        switch p.curToken.Type {
        case token.MODEL, token.SERVICE, token.RAW_GO, token.RAW_TEMPLATE:
            return
        }
        if p.curTokenIs(token.RBRACE) {
            p.nextToken()
            return
        }
        p.nextToken()
    }
}
```

**Résultat** : Le parser peut continuer même après une erreur, pour afficher **toutes** les erreurs d'un coup.

### Script Parser Fallback

Si le parsing du script échoue, le `ScriptBlock` garde le source brut :

```go
scriptBlock := &ast.ScriptBlock{
    Source:    source,      // ✅ Toujours présent
    Funcs:     funcs,       // nil si erreur
    StartLine: lineOffset,
}
```

Le generator peut alors utiliser le fallback.

## Optimisations Possibles

Voir `AUDIT_REPORT.md` pour les duplications identifiées :

1. **genRouteRegistry** appelé 3 fois → 1 seul appel
2. **needsXxxHelper** répété 4 fois → généraliser
3. **toPascalCase** dupliqué → package utils
4. **Regex compilation** à chaque appel → compiler une fois

## Métriques

| Package | LOC | Complexité | Couverture |
|---------|-----|------------|------------|
| lexer | 466 | Faible | 87.7% |
| parser | 289 | Faible | 86.6% |
| script/parser | 790 | Moyenne | 72.6% |
| script/transpiler | 625 | Moyenne | 72.6% |
| generator | 915 | Élevée | 78.5% |

**Total** : ~3085 LOC de logique de compilation (hors tests)

## Prochaines Étapes

- **[AST](ast.md)** — Structures de données AST
- **[Lexer & Parser](lexer-parser.md)** — Détails du parsing
- **[Generator](generator.md)** — Détails de la génération
- **[Script Transpiler](script-transpiler.md)** — Transpilation GMX → Go
- **[Testing](testing.md)** — Stratégie de test
