# Script Transpiler

Le transpiler convertit les fonctions GMX Script (TypeScript-inspired) en fonctions Go idiomatiques.

## Architecture

**Fichiers** :
- `internal/compiler/script/parser.go` — Parse GMX Script en AST
- `internal/compiler/script/transpiler.go` — Transpile AST en Go

## Flow

```
GMX Script → Parser → AST → Transpiler → Go Code
```

## Transpiler Structure

```go
type Transpiler struct {
    buf         strings.Builder
    sourceMap   *SourceMap
    goLine      int
    indent      int
    models      []string
    errDeclared bool
    varTypes    map[string]string
    currentFunc string
}
```

## Entrée : Transpile

```go
func Transpile(script *ast.ScriptBlock, modelNames []string) *TranspileResult {
    t := NewTranspiler(modelNames)

    // 1. Generate ORM helpers
    t.genORMHelpers()

    // 2. Generate GMXContext struct
    t.genGMXContext()

    // 3. Generate renderFragment helper
    t.genRenderFragment()

    // 4. Transpile each function
    for _, fn := range script.Funcs {
        t.TranspileFunc(fn)
    }

    return &TranspileResult{
        GoCode:    t.buf.String(),
        SourceMap: t.sourceMap,
        Errors:    []string{},
    }
}
```

## Transpilation de Fonction

### Signature

**GMX** :

```gmx
func toggleTask(id: uuid) error
```

**Go** :

```go
func toggleTask(ctx *GMXContext, id string) error
```

**Code** :

```go
func (t *Transpiler) TranspileFunc(fn *ast.FuncDecl) {
    t.emit("func %s(ctx *GMXContext", fn.Name)

    for _, param := range fn.Params {
        t.emit(", %s %s", param.Name, t.transpileType(param.Type))
    }

    t.emit(") error {\n")
    t.indent++

    for _, stmt := range fn.Body {
        t.transpileStmt(stmt)
    }

    if !t.endsWithReturn(fn.Body) {
        t.emitIndent()
        t.emit("return nil\n")
    }

    t.indent--
    t.emit("}\n")
}
```

## Transpilation de Statements

### LetStmt

**GMX** :

```gmx
let task = try Task.find(id)
```

**Go** :

```go
task, err := TaskFind(ctx.DB, id)
if err != nil {
    return err
}
```

**Code** :

```go
func (t *Transpiler) transpileLetStmt(stmt *ast.LetStmt) {
    if tryExpr, ok := stmt.Value.(*ast.TryExpr); ok {
        if t.errDeclared {
            t.emit("%s, err = %s\n", stmt.Name, t.transpileExpr(tryExpr.Expr))
        } else {
            t.emit("%s, err := %s\n", stmt.Name, t.transpileExpr(tryExpr.Expr))
            t.errDeclared = true
        }
        t.emit("if err != nil {\n\treturn err\n}\n")
    } else {
        t.emit("%s := %s\n", stmt.Name, t.transpileExpr(stmt.Value))
    }
}
```

### AssignStmt

**GMX** :

```gmx
task.done = !task.done
```

**Go** :

```go
task.Done = !task.Done
```

**Code** :

```go
func (t *Transpiler) transpileAssignStmt(stmt *ast.AssignStmt) {
    target := t.transpileExpr(stmt.Target)
    value := t.transpileExpr(stmt.Value)
    t.emit("%s = %s\n", target, value)
}
```

### ReturnStmt

**GMX** :

```gmx
return render(task)
```

**Go** :

```go
return renderFragment(ctx.Writer, "task", task)
```

**Code** :

```go
func (t *Transpiler) transpileReturnStmt(stmt *ast.ReturnStmt) {
    if stmt.Value == nil {
        t.emit("return nil\n")
    } else {
        t.emit("return %s\n", t.transpileExpr(stmt.Value))
    }
}
```

### IfStmt

**GMX** :

```gmx
if title == "" {
    return error("Title cannot be empty")
}
```

**Go** :

```go
if title == "" {
    return fmt.Errorf("Title cannot be empty")
}
```

**Code** :

```go
func (t *Transpiler) transpileIfStmt(stmt *ast.IfStmt) {
    t.emit("if %s {\n", t.transpileExpr(stmt.Condition))
    t.indent++
    for _, s := range stmt.Consequence {
        t.transpileStmt(s)
    }
    t.indent--

    if len(stmt.Alternative) > 0 {
        t.emit("} else {\n")
        t.indent++
        for _, s := range stmt.Alternative {
            t.transpileStmt(s)
        }
        t.indent--
    }

    t.emit("}\n")
}
```

## Transpilation d'Expressions

### CallExpr (ORM Methods)

**GMX** :

```gmx
Task.find(id)
```

**Go** :

```go
TaskFind(ctx.DB, id)
```

**Code** :

```go
func (t *Transpiler) transpileCallExpr(expr *ast.CallExpr) string {
    if memberExpr, ok := expr.Function.(*ast.MemberExpr); ok {
        if objIdent, ok := memberExpr.Object.(*ast.Ident); ok {
            // Check if it's a model ORM method
            if t.isModel(objIdent.Name) {
                switch memberExpr.Property {
                case "find":
                    return fmt.Sprintf("%sFind(ctx.DB, %s)",
                        objIdent.Name, t.transpileArgs(expr.Args))
                case "all":
                    return fmt.Sprintf("%sAll(ctx.DB)", objIdent.Name)
                case "save":
                    return fmt.Sprintf("%sSave(ctx.DB, %s)",
                        objIdent.Name, objIdent.Name)
                case "delete":
                    return fmt.Sprintf("%sDelete(ctx.DB, %s)",
                        objIdent.Name, objIdent.Name)
                }
            }
        }
    }

    // Regular function call
    return fmt.Sprintf("%s(%s)",
        t.transpileExpr(expr.Function),
        t.transpileArgs(expr.Args))
}
```

### MemberExpr

**GMX** :

```gmx
task.title
```

**Go** :

```go
task.Title
```

**Code** :

```go
func (t *Transpiler) transpileMemberExpr(expr *ast.MemberExpr) string {
    obj := t.transpileExpr(expr.Object)
    prop := t.toPascalCase(expr.Property)
    return fmt.Sprintf("%s.%s", obj, prop)
}
```

### RenderExpr

**GMX** :

```gmx
render(task)
```

**Go** :

```go
renderFragment(ctx.Writer, "task", task)
```

**Code** :

```go
func (t *Transpiler) transpileRenderExpr(expr *ast.RenderExpr) string {
    if len(expr.Args) == 0 {
        return "renderFragment(ctx.Writer, \"default\", nil)"
    }
    if len(expr.Args) == 1 {
        arg := t.transpileExpr(expr.Args[0])
        return fmt.Sprintf("renderFragment(ctx.Writer, \"fragment\", %s)", arg)
    }
    // Multiple args
    return fmt.Sprintf("renderFragment(ctx.Writer, \"combined\", map[string]interface{}{...})")
}
```

### ErrorExpr

**GMX** :

```gmx
error("Title cannot be empty")
```

**Go** :

```go
fmt.Errorf("Title cannot be empty")
```

**Code** :

```go
func (t *Transpiler) transpileErrorExpr(expr *ast.ErrorExpr) string {
    msg := t.transpileExpr(expr.Message)
    return fmt.Sprintf("fmt.Errorf(%s)", msg)
}
```

### TryExpr

**GMX** :

```gmx
try Task.find(id)
```

**Transpilé dans le contexte** :

```go
task, err := TaskFind(ctx.DB, id)
if err != nil {
    return err
}
```

Le `try` est géré par `transpileLetStmt` ou `transpileExprStmt`, pas directement.

## ORM Helpers

Le transpiler génère automatiquement ces helpers pour chaque modèle :

```go
func TaskFind(db *gorm.DB, id string) (*Task, error) {
    var obj Task
    if err := db.First(&obj, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &obj, nil
}

func TaskAll(db *gorm.DB) ([]Task, error) {
    var objs []Task
    if err := db.Find(&objs).Error; err != nil {
        return nil, err
    }
    return objs, nil
}

func TaskSave(db *gorm.DB, obj *Task) error {
    if err := obj.Validate(); err != nil {
        return err
    }
    return db.Save(obj).Error
}

func TaskDelete(db *gorm.DB, obj *Task) error {
    return db.Delete(obj).Error
}
```

## GMXContext

```go
type GMXContext struct {
    DB      *gorm.DB
    Tenant  string
    User    string
    Writer  http.ResponseWriter
    Request *http.Request
}
```

Injecté automatiquement comme premier paramètre de chaque fonction.

## Source Maps

Le transpiler maintient un mapping ligne GMX → ligne Go :

```go
type SourceMapEntry struct {
    GoLine  int
    GmxLine int
    GmxFile string
}
```

**Usage** : Afficher les erreurs Go avec les numéros de ligne GMX originaux.

## Exemple Complet

**GMX** :

```gmx
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}
```

**Go Généré** :

```go
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

## Limitations

### Interpolation de Chaînes

**GMX** :

```gmx
let msg = "Hello, {name}!"
```

**Go** :

```go
msg := fmt.Sprintf("Hello, %s!", name)
```

**Bug connu** : L'interpolation avec membre access (`{task.title}`) ne fonctionne pas toujours.

### Boucles

Pas encore implémenté. Workaround : utiliser Go directement dans le code généré.

### Switch/Case

Pas encore implémenté. Utiliser `if/else`.

## Prochaines Étapes

- **[Testing](testing.md)** — Tests du transpiler
