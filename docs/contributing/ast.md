# AST (Abstract Syntax Tree)

L'AST de GMX représente la structure complète d'un fichier `.gmx` parsé. Tous les types sont définis dans `internal/compiler/ast/ast.go`.

## Structure Racine

### GMXFile

```go
type GMXFile struct {
    Models   []*ModelDecl
    Services []*ServiceDecl
    Script   *ScriptBlock
    Template *TemplateBlock
    Style    *StyleBlock
}
```

C'est le nœud racine qui contient toutes les sections d'un fichier `.gmx`.

## Section Models

### ModelDecl

```go
type ModelDecl struct {
    Name   string
    Fields []*FieldDecl
}
```

Représente `model Task { ... }`.

### FieldDecl

```go
type FieldDecl struct {
    Name        string
    Type        string
    Annotations []*Annotation
}
```

Représente `title: string @min(3) @max(255)`.

**Types supportés** : `uuid`, `string`, `int`, `float`, `bool`, `datetime`, ou nom de modèle (relation).

### Annotation

```go
type Annotation struct {
    Name string            // "pk", "default", "min", "email", etc.
    Args map[string]string // Arguments nommés ou simples
}
```

**Exemples** :

| GMX | Name | Args |
|-----|------|------|
| `@pk` | `"pk"` | `{}` |
| `@default(false)` | `"default"` | `{"_": "false"}` |
| `@min(3)` | `"min"` | `{"_": "3"}` |
| `@relation(references: [id])` | `"relation"` | `{"references": "id"}` |

**Helper** : `SimpleArg()` retourne la valeur pour une annotation simple.

```go
ann := &Annotation{Name: "min", Args: {"_": "3"}}
ann.SimpleArg()  // → "3"
```

## Section Services

### ServiceDecl

```go
type ServiceDecl struct {
    Name     string
    Provider string
    Fields   []*ServiceField
    Methods  []*ServiceMethod
}
```

Représente `service Database { provider: "sqlite"; ... }`.

### ServiceField

```go
type ServiceField struct {
    Name        string
    Type        string
    EnvVar      string
    Annotations []*Annotation
}
```

Représente `url: string @env("DATABASE_URL")`.

### ServiceMethod

```go
type ServiceMethod struct {
    Name       string
    Params     []*Param
    ReturnType string
}
```

Représente `func send(to: string, subject: string, body: string) error`.

## Section Script

### ScriptBlock

```go
type ScriptBlock struct {
    Source    string       // Raw source (fallback)
    Funcs     []*FuncDecl  // Parsed functions (nil if parsing failed)
    StartLine int          // Line offset for source maps
}
```

### FuncDecl

```go
type FuncDecl struct {
    Name       string
    Params     []*Param
    ReturnType string
    Body       []Statement
    Line       int
}
```

Représente `func toggleTask(id: uuid) error { ... }`.

### Param

```go
type Param struct {
    Name string
    Type string
}
```

## Statements

Tous les statements implémentent l'interface :

```go
type Statement interface {
    Node
    statementNode()
}
```

### LetStmt

```go
type LetStmt struct {
    Name  string
    Value Expression
    Const bool  // true si 'const'
    Line  int
}
```

Représente `let task = try Task.find(id)` ou `const x = 5`.

### AssignStmt

```go
type AssignStmt struct {
    Target Expression  // Ident ou MemberExpr
    Value  Expression
    Line   int
}
```

Représente `task.done = !task.done`.

### ReturnStmt

```go
type ReturnStmt struct {
    Value Expression  // nil pour 'return'
    Line  int
}
```

### IfStmt

```go
type IfStmt struct {
    Condition   Expression
    Consequence []Statement
    Alternative []Statement  // nil si pas de 'else'
    Line        int
}
```

### ExprStmt

```go
type ExprStmt struct {
    Expr Expression
    Line int
}
```

Utilisé pour les appels de fonction utilisés comme statements.

## Expressions

Toutes les expressions implémentent :

```go
type Expression interface {
    Node
    expressionNode()
}
```

### Ident

```go
type Ident struct {
    Name string
    Line int
}
```

Représente une variable : `task`, `count`, etc.

### Littéraux

```go
type IntLit struct {
    Value string
    Line  int
}

type FloatLit struct {
    Value string
    Line  int
}

type StringLit struct {
    Value string
    Parts []StringPart  // Pour interpolation
    Line  int
}

type BoolLit struct {
    Value bool
    Line  int
}
```

### StringPart (Interpolation)

```go
type StringPart struct {
    IsExpr bool
    Text   string      // Si !IsExpr
    Expr   Expression  // Si IsExpr
}
```

Pour `"Hello, {name}!"` :

```go
Parts: []StringPart{
    {IsExpr: false, Text: "Hello, "},
    {IsExpr: true, Expr: &Ident{Name: "name"}},
    {IsExpr: false, Text: "!"},
}
```

### UnaryExpr

```go
type UnaryExpr struct {
    Op      string  // "!", "-"
    Operand Expression
    Line    int
}
```

Représente `!task.done` ou `-count`.

### BinaryExpr

```go
type BinaryExpr struct {
    Left  Expression
    Op    string  // "+", "==", "&&", etc.
    Right Expression
    Line  int
}
```

Représente `count + 1`, `task.done == true`, etc.

### CallExpr

```go
type CallExpr struct {
    Function Expression  // Ident ou MemberExpr
    Args     []Expression
    Line     int
}
```

Représente `Task.find(id)` ou `processData(x, y)`.

### MemberExpr

```go
type MemberExpr struct {
    Object   Expression
    Property string
    Line     int
}
```

Représente `task.title`, `user.email`, `Task.find`, etc.

### TryExpr

```go
type TryExpr struct {
    Expr Expression
    Line int
}
```

Représente `try Task.find(id)`.

### RenderExpr

```go
type RenderExpr struct {
    Args []Expression
    Line int
}
```

Représente `render(task)` ou `render(task, user)`.

### ErrorExpr

```go
type ErrorExpr struct {
    Message Expression
    Line    int
}
```

Représente `error("Title cannot be empty")`.

### CtxExpr

```go
type CtxExpr struct {
    Field string
    Line  int
}
```

Représente `ctx.User`, `ctx.Tenant`, etc.

### StructLit

```go
type StructLit struct {
    TypeName string
    Fields   map[string]Expression
    Line     int
}
```

Représente `Task{title: "New", done: false}`.

## Section Template

### TemplateBlock

```go
type TemplateBlock struct {
    Source string  // Raw HTML template
}
```

Contient le HTML brut avec syntaxe Go template.

## Section Style

### StyleBlock

```go
type StyleBlock struct {
    Source string  // Raw CSS
    Scoped bool    // true si '<style scoped>'
}
```

## Interface Node

Tous les nœuds implémentent :

```go
type Node interface {
    TokenLiteral() string
}
```

Utilisé pour le debugging et les messages d'erreur.

## Exemple Complet

**GMX** :

```gmx
model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3)
  done:  bool
}

<script>
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}
</script>
```

**AST Partiel** :

```go
&ast.GMXFile{
    Models: []*ast.ModelDecl{
        {
            Name: "Task",
            Fields: []*ast.FieldDecl{
                {
                    Name: "id",
                    Type: "uuid",
                    Annotations: []*ast.Annotation{
                        {Name: "pk", Args: map[string]string{}},
                        {Name: "default", Args: map[string]string{"_": "uuid_v4"}},
                    },
                },
                {
                    Name: "title",
                    Type: "string",
                    Annotations: []*ast.Annotation{
                        {Name: "min", Args: map[string]string{"_": "3"}},
                    },
                },
                {Name: "done", Type: "bool", Annotations: nil},
            },
        },
    },
    Script: &ast.ScriptBlock{
        Source: "func toggleTask(id: uuid) error { ... }",
        Funcs: []*ast.FuncDecl{
            {
                Name: "toggleTask",
                Params: []*ast.Param{
                    {Name: "id", Type: "uuid"},
                },
                ReturnType: "error",
                Body: []ast.Statement{
                    &ast.LetStmt{
                        Name: "task",
                        Value: &ast.TryExpr{
                            Expr: &ast.CallExpr{
                                Function: &ast.MemberExpr{
                                    Object: &ast.Ident{Name: "Task"},
                                    Property: "find",
                                },
                                Args: []ast.Expression{
                                    &ast.Ident{Name: "id"},
                                },
                            },
                        },
                    },
                    &ast.AssignStmt{
                        Target: &ast.MemberExpr{
                            Object: &ast.Ident{Name: "task"},
                            Property: "done",
                        },
                        Value: &ast.UnaryExpr{
                            Op: "!",
                            Operand: &ast.MemberExpr{
                                Object: &ast.Ident{Name: "task"},
                                Property: "done",
                            },
                        },
                    },
                    &ast.ExprStmt{
                        Expr: &ast.TryExpr{
                            Expr: &ast.CallExpr{
                                Function: &ast.MemberExpr{
                                    Object: &ast.Ident{Name: "task"},
                                    Property: "save",
                                },
                                Args: []ast.Expression{},
                            },
                        },
                    },
                    &ast.ReturnStmt{
                        Value: &ast.RenderExpr{
                            Args: []ast.Expression{
                                &ast.Ident{Name: "task"},
                            },
                        },
                    },
                },
            },
        },
    },
}
```

## Prochaines Étapes

- **[Lexer & Parser](lexer-parser.md)** — Comment l'AST est construit
- **[Generator](generator.md)** — Comment l'AST est utilisé
- **[Script Transpiler](script-transpiler.md)** — Transpilation de l'AST script
