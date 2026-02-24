# GMX Script

GMX Script est un langage inspiré de TypeScript qui se transpile en Go. Il permet d'écrire la logique métier de manière concise avec gestion d'erreurs automatique et méthodes ORM intégrées.

## Syntaxe de Base

```gmx
<script>
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}
</script>
```

**Transpilé en Go :**

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

## Déclarations de Variables

### `let` — Variable Mutable

```gmx
let count = 0
count = count + 1
```

Transpilé :

```go
count := 0
count = count + 1
```

### `const` — Variable Immutable

```gmx
const maxRetries = 3
```

Transpilé :

```go
maxRetries := 3
```

!!!note "Différence avec TypeScript"
    En GMX, `const` génère simplement un `:=` en Go. L'immutabilité n'est pas forcée par le compilateur Go.

## Types

| GMX Type | Go Type   | Usage                          |
|----------|-----------|--------------------------------|
| `string` | `string`  | Texte                          |
| `int`    | `int`     | Nombres entiers                |
| `float`  | `float64` | Nombres décimaux               |
| `bool`   | `bool`    | Vrai/faux                      |
| `uuid`   | `string`  | Identifiants (transpilé)       |
| `error`  | `error`   | Type de retour obligatoire     |

### Types de Modèles

Les modèles GMX sont utilisés comme types :

```gmx
func processTask(t: Task) error {
  let title = t.title
  return nil
}
```

Transpilé :

```go
func processTask(ctx *GMXContext, t *Task) error {
    title := t.Title
    return nil
}
```

## Gestion des Erreurs

### `try` — Unwrap ou Return

```gmx
let task = try Task.find(id)
```

**Transpilé en :**

```go
task, err := TaskFind(ctx.DB, id)
if err != nil {
    return err
}
```

Le `try` **unwrap automatiquement** et **return l'erreur** si elle existe.

### `error()` — Créer une Erreur

```gmx
if title == "" {
  return error("Title cannot be empty")
}
```

Transpilé :

```go
if title == "" {
    return fmt.Errorf("Title cannot be empty")
}
```

### Toutes les Fonctions Retournent `error`

```gmx
func deleteTask(id: uuid) error {
  let task = try Task.find(id)
  try task.delete()
  return nil  // ✅ Success
}
```

**IMPORTANT** : Le type de retour `error` est **obligatoire** pour toutes les fonctions GMX Script.

## Méthodes ORM

### `Model.find(id)`

Trouve une entité par son ID :

```gmx
let task = try Task.find(taskId)
```

Transpilé :

```go
task, err := TaskFind(ctx.DB, taskId)
if err != nil {
    return err
}
```

### `Model.all()`

Récupère toutes les entités :

```gmx
let tasks = try Task.all()
```

Transpilé :

```go
tasks, err := TaskAll(ctx.DB)
if err != nil {
    return err
}
```

### `instance.save()`

Crée ou met à jour une entité :

```gmx
const task = Task{title: "New task", done: false}
try task.save()
```

Transpilé :

```go
task := &Task{Title: "New task", Done: false}
if err := TaskSave(ctx.DB, task); err != nil {
    return err
}
```

### `instance.delete()`

Supprime une entité :

```gmx
let task = try Task.find(id)
try task.delete()
```

Transpilé :

```go
task, err := TaskFind(ctx.DB, id)
if err != nil {
    return err
}
if err := TaskDelete(ctx.DB, task); err != nil {
    return err
}
```

## Rendu de Templates

### `render(data)`

Rend un fragment de template avec des données :

```gmx
return render(task)
```

Transpilé :

```go
return renderFragment(ctx.Writer, "task", task)
```

### `render()` Multiple

Vous pouvez passer plusieurs arguments :

```gmx
return render(task, user, posts)
```

Transpilé :

```go
data := map[string]interface{}{
    "task": task,
    "user": user,
    "posts": posts,
}
return renderFragment(ctx.Writer, "combined", data)
```

## Structures de Contrôle

### `if / else`

```gmx
if task.done {
  return error("Task already completed")
} else {
  task.done = true
}
```

Transpilé :

```go
if task.Done {
    return fmt.Errorf("Task already completed")
} else {
    task.Done = true
}
```

### Conditions Complexes

```gmx
if count > 10 && status == "active" {
  // ...
}
```

Opérateurs supportés :
- **Comparaison** : `==`, `!=`, `<`, `>`, `<=`, `>=`
- **Logique** : `&&`, `||`, `!`
- **Arithmétique** : `+`, `-`, `*`, `/`, `%`

## Expressions

### Opérateurs Binaires

```gmx
let total = price * quantity
let isValid = count > 0 && count < 100
```

### Opérateurs Unaires

```gmx
let isNotDone = !task.done
let negative = -amount
```

### Accès aux Membres

```gmx
let title = task.title
let userEmail = post.author.email
```

Transpilé en PascalCase :

```go
title := task.Title
userEmail := post.Author.Email
```

### Appels de Fonctions

```gmx
let result = processData(input, options)
```

### Littéraux de Structures

```gmx
const task = Task{
  title: "Buy milk",
  done: false
}
```

Transpilé :

```go
task := &Task{
    Title: "Buy milk",
    Done: false,
}
```

## Interpolation de Chaînes

```gmx
let message = "Hello, {user.name}!"
```

Transpilé :

```go
message := fmt.Sprintf("Hello, %s!", user.Name)
```

**Expressions supportées** :

```gmx
let msg = "Count: {count + 1}"
let msg = "Status: {task.done ? "done" : "pending"}"
```

!!!warning "Limitation Actuelle"
    L'interpolation avec accès aux membres (`{task.title}`) a des bugs connus. Préférez :
    ```gmx
    let title = task.title
    let msg = "Title: {title}"
    ```

## Contexte Implicite

### `ctx` — Contexte de Requête

Le contexte HTTP est toujours disponible :

```gmx
func getCurrentUser() error {
  let userId = ctx.User
  let tenantId = ctx.Tenant
  // ...
}
```

**Champs disponibles** :

```go
type GMXContext struct {
    DB      *gorm.DB
    Tenant  string
    User    string
    Writer  http.ResponseWriter
    Request *http.Request
}
```

## Exemples Complets

### CRUD Simple

```gmx
<script>
func createTask(title: string) error {
  if title == "" {
    return error("Title cannot be empty")
  }

  const task = Task{title: title, done: false}
  try task.save()
  return render(task)
}

func listTasks() error {
  let tasks = try Task.all()
  return render(tasks)
}

func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}

func deleteTask(id: uuid) error {
  let task = try Task.find(id)
  try task.delete()
  return nil
}
</script>
```

### Validation Métier

```gmx
<script>
func createPost(title: string, content: string) error {
  if title == "" {
    return error("Title is required")
  }

  if len(content) < 50 {
    return error("Content must be at least 50 characters")
  }

  const post = Post{
    title: title,
    content: content,
    authorId: ctx.User
  }

  try post.save()
  return render(post)
}
</script>
```

### Relations

```gmx
<script>
func getUserWithPosts(userId: uuid) error {
  let user = try User.find(userId)
  let posts = try Post.all()  // TODO: filter by userId

  return render(user, posts)
}
</script>
```

!!!note "Filtrage Personnalisé"
    Actuellement, `Model.all()` récupère tout. Pour filtrer, utilisez GORM directement dans le code généré (limitation temporaire).

## Transpilation Détaillée

### Fonction Minimale

**GMX :**

```gmx
func hello(name: string) error {
  return error("Hello, {name}")
}
```

**Go généré :**

```go
func hello(ctx *GMXContext, name string) error {
    return fmt.Errorf("Hello, %s", name)
}
```

### Avec ORM

**GMX :**

```gmx
func getTask(id: uuid) error {
  let task = try Task.find(id)
  return render(task)
}
```

**Go généré :**

```go
func getTask(ctx *GMXContext, id string) error {
    task, err := TaskFind(ctx.DB, id)
    if err != nil {
        return err
    }
    return renderFragment(ctx.Writer, "task", task)
}
```

### Helpers Générés

Le transpiler génère automatiquement ces helpers :

```go
// ORM Helpers
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

// GMXContext
type GMXContext struct {
    DB      *gorm.DB
    Tenant  string
    User    string
    Writer  http.ResponseWriter
    Request *http.Request
}

// Render Helper
func renderFragment(w http.ResponseWriter, name string, data interface{}) error {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    return tmpl.ExecuteTemplate(w, name, data)
}
```

## Limitations Actuelles

| Fonctionnalité | Status |
|----------------|--------|
| Variables (let/const) | ✅ Implémenté |
| try/error | ✅ Implémenté |
| if/else | ✅ Implémenté |
| Opérateurs (==, !=, &&, etc.) | ✅ Implémenté |
| ORM methods (find, all, save, delete) | ✅ Implémenté |
| render() | ✅ Implémenté |
| Interpolation simple | ✅ Implémenté |
| Interpolation avec membres | 🟡 Buggy |
| for loops | ❌ Non implémenté |
| switch/case | ❌ Non implémenté |
| Fonctions anonymes | ❌ Non implémenté |
| async/await | ❌ Non implémenté |

## Bonnes Pratiques

### ✅ Do

- Toujours retourner `error`
- Utiliser `try` pour les appels ORM
- Valider les inputs avant `save()`
- Nommer les fonctions en camelCase
- Garder les fonctions courtes (< 20 lignes)

### ❌ Don't

- Ne pas oublier `try` sur les méthodes ORM
- Ne pas ignorer les erreurs
- Ne pas faire de logique complexe (utiliser Go pur dans le code généré)
- Ne pas utiliser des boucles (pas encore supporté)

## Debugging

### Voir le Code Go Généré

```bash
gmx app.gmx main.go
cat main.go | grep -A 20 "func toggleTask"
```

### Erreurs de Transpilation

Si le transpiler échoue, le compiler affiche :

```
Generation Error: transpile errors: [...]
```

**Solutions courantes** :

1. Vérifier que toutes les fonctions retournent `error`
2. Vérifier la syntaxe des `try` statements
3. S'assurer que les types de modèles existent

## Comparaison GMX ↔ Go

### Variable Declaration

| GMX | Go |
|-----|-----|
| `let x = 5` | `x := 5` |
| `const x = 5` | `x := 5` |

### Error Handling

| GMX | Go |
|-----|-----|
| `let x = try f()` | `x, err := f()`<br>`if err != nil { return err }` |
| `return error("msg")` | `return fmt.Errorf("msg")` |

### ORM Methods

| GMX | Go |
|-----|-----|
| `Task.find(id)` | `TaskFind(ctx.DB, id)` |
| `Task.all()` | `TaskAll(ctx.DB)` |
| `task.save()` | `TaskSave(ctx.DB, task)` |
| `task.delete()` | `TaskDelete(ctx.DB, task)` |

### Rendering

| GMX | Go |
|-----|-----|
| `render(task)` | `renderFragment(ctx.Writer, "task", task)` |

## Prochaines Étapes

- **[Templates](templates.md)** — Connecter le script aux templates HTMX
- **[Security](security.md)** — Validation et sécurité dans le script
- **[Contributing](../contributing/script-transpiler.md)** — Architecture du transpiler
