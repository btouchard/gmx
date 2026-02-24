# Templates

Les templates GMX combinent Go templates et HTMX pour créer des interfaces dynamiques avec validation des routes au compile-time.

## Structure de Base

```gmx
<template>
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>My App</title>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
  <h1>Hello, GMX!</h1>
</body>
</html>
</template>
```

## Syntaxe Go Templates

GMX utilise le package `html/template` de Go. Voici les constructions principales :

### Variables

```html
<h1>{{.Title}}</h1>
<p>User ID: {{.UserID}}</p>
```

### Conditions

```html
{{if .Done}}
  <span class="done">✓ Completed</span>
{{else}}
  <span class="pending">Pending</span>
{{end}}
```

### Boucles

```html
<ul>
  {{range .Tasks}}
    <li>{{.Title}} - {{if .Done}}Done{{else}}Todo{{end}}</li>
  {{end}}
</ul>
```

### Comparaisons

```html
{{if eq .Status "active"}}Active{{end}}
{{if ne .Count 0}}Count: {{.Count}}{{end}}
{{if gt .Score 100}}High score!{{end}}
```

Opérateurs disponibles : `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `and`, `or`, `not`

## Routes HTMX

### `{{route "functionName"}}` — Route Helper

GMX génère automatiquement des routes depuis les fonctions script :

```gmx
<script>
func toggleTask(id: uuid) error {
  // ...
}
</script>

<template>
<button hx-patch="{{route "toggleTask"}}?id={{.ID}}">
  Toggle
</button>
</template>
```

**Génère** :

```go
// Route registry
routes := map[string]string{
    "toggleTask": "/toggleTask",
}

// Template function
funcMap := template.FuncMap{
    "route": func(name string) string {
        if path, ok := routes[name]; ok {
            return path
        }
        return "#unknown-route"
    },
}
```

**IMPORTANT** : Les routes sont **validées au compile-time**. Si `toggleTask` n'existe pas dans le script, la génération échoue.

### Routes Avec Paramètres

```html
<a href="{{route "viewPost"}}?id={{.PostID}}">View</a>

<button
  hx-delete="{{route "deleteTask"}}?id={{.TaskID}}"
  hx-confirm="Delete this task?">
  Delete
</button>
```

## HTMX Integration

### Attributs HTMX

```html
<!-- GET -->
<button hx-get="{{route "loadMore"}}" hx-target="#results">
  Load More
</button>

<!-- POST -->
<form hx-post="{{route "createTask"}}" hx-target="#task-list" hx-swap="beforeend">
  <input type="text" name="title" required />
  <button type="submit">Create</button>
</form>

<!-- PATCH -->
<input
  type="checkbox"
  hx-patch="{{route "toggleTask"}}?id={{.ID}}"
  hx-target="closest .task-item"
  hx-swap="outerHTML" />

<!-- DELETE -->
<button
  hx-delete="{{route "deleteTask"}}?id={{.ID}}"
  hx-target="closest .task-item"
  hx-swap="outerHTML swap:1s">
  Delete
</button>
```

### Swap Strategies

```html
<!-- Replace inner HTML -->
<div hx-get="/endpoint" hx-swap="innerHTML"></div>

<!-- Replace outer HTML -->
<div hx-get="/endpoint" hx-swap="outerHTML"></div>

<!-- Append at end -->
<div hx-get="/endpoint" hx-swap="beforeend"></div>

<!-- Insert before -->
<div hx-get="/endpoint" hx-swap="beforebegin"></div>

<!-- With animation delay -->
<div hx-get="/endpoint" hx-swap="outerHTML swap:1s"></div>
```

### Target Selectors

```html
<!-- Target by ID -->
<button hx-get="/data" hx-target="#results">Load</button>

<!-- Target closest parent -->
<button hx-delete="/delete" hx-target="closest .item">Delete</button>

<!-- Target this element -->
<button hx-get="/toggle" hx-target="this">Toggle</button>
```

## Data Binding

### PageData Struct

GMX génère automatiquement une structure `PageData` avec tous les modèles :

```gmx
model Task { ... }
model User { ... }
```

**Génère** :

```go
type PageData struct {
    CSRFToken string
    Tasks     []Task
    Users     []User
}
```

### Accès aux Données

```html
<h2>Tasks ({{len .Tasks}})</h2>
<ul>
  {{range .Tasks}}
    <li id="task-{{.ID}}">
      <span>{{.Title}}</span>
      {{if .Done}}<strong>✓</strong>{{end}}
    </li>
  {{end}}
</ul>
```

### CSRF Token

Le token CSRF est **toujours disponible** :

```html
<form method="POST" action="/submit">
  <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />
  <!-- ... -->
</form>
```

**IMPORTANT** : Avec HTMX, le token est **injecté automatiquement** dans les headers. Pas besoin de champ hidden.

## Fragments et Layouts

### Template Named Blocks

```html
{{define "task-item"}}
<li class="task-item {{if .Done}}done{{end}}" id="task-{{.ID}}">
  <input
    type="checkbox"
    {{if .Done}}checked{{end}}
    hx-patch="{{route "toggleTask"}}?id={{.ID}}"
    hx-target="#task-{{.ID}}"
    hx-swap="outerHTML" />
  <span>{{.Title}}</span>
</li>
{{end}}
```

**Utilisation** :

```html
<ul id="task-list">
  {{range .Tasks}}
    {{template "task-item" .}}
  {{end}}
</ul>
```

### Render depuis le Script

```gmx
<script>
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)  // ← Rend "task-item" template
}
</script>
```

**Génère** :

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

func renderFragment(w http.ResponseWriter, name string, data interface{}) error {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    return tmpl.ExecuteTemplate(w, name, data)
}
```

## Exemple Complet

```gmx
model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3) @max(255)
  done:  bool   @default(false)
}

<script>
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}

func createTask(title: string) error {
  if title == "" {
    return error("Title cannot be empty")
  }
  const task = Task{title: title, done: false}
  try task.save()
  return render(task)
}

func deleteTask(id: uuid) error {
  let task = try Task.find(id)
  try task.delete()
  return nil
}
</script>

<template>
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Todo App</title>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
  <style>
    body { font-family: sans-serif; max-width: 800px; margin: 2rem auto; padding: 1rem; }
    .task-item { padding: 1rem; border-bottom: 1px solid #eee; display: flex; align-items: center; gap: 1rem; }
    .task-item.done .task-title { text-decoration: line-through; opacity: 0.6; }
    .task-delete { background: #dc3545; color: white; border: none; padding: 0.5rem 1rem; border-radius: 4px; cursor: pointer; }
  </style>
</head>
<body>
  <h1>📝 Todo App</h1>

  <form hx-post="{{route "createTask"}}" hx-target="#task-list" hx-swap="beforeend">
    <input type="text" name="title" placeholder="What needs to be done?" required />
    <button type="submit">Add Task</button>
  </form>

  <ul id="task-list">
    {{range .Tasks}}
    <li class="task-item {{if .Done}}done{{end}}" id="task-{{.ID}}">
      <input
        type="checkbox"
        {{if .Done}}checked{{end}}
        hx-patch="{{route "toggleTask"}}?id={{.ID}}"
        hx-target="#task-{{.ID}}"
        hx-swap="outerHTML" />
      <span class="task-title">{{.Title}}</span>
      <button
        class="task-delete"
        hx-delete="{{route "deleteTask"}}?id={{.ID}}"
        hx-target="#task-{{.ID}}"
        hx-swap="outerHTML swap:1s">
        Delete
      </button>
    </li>
    {{end}}
  </ul>
</body>
</html>
</template>
```

## Security Headers

GMX génère automatiquement des security headers :

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
```

## CSRF Protection

### Double Submit Cookie

GMX implémente le pattern "double submit cookie" :

1. **GET request** : Set cookie + inject token
2. **POST/PATCH/DELETE** : Validate cookie == header

```go
// GET: Set CSRF cookie
csrfToken := generateCSRFToken()
http.SetCookie(w, &http.Cookie{
    Name:     "csrf_token",
    Value:    csrfToken,
    HttpOnly: true,
    SameSite: http.SameSiteStrictMode,
})

// POST/PATCH/DELETE: Validate
cookie, _ := r.Cookie("csrf_token")
headerToken := r.Header.Get("X-CSRF-Token")
if cookie.Value != headerToken {
    http.Error(w, "CSRF validation failed", http.StatusForbidden)
    return
}
```

### HTMX CSRF Injection

GMX injecte automatiquement le token dans les requêtes HTMX :

```html
<script>
document.body.addEventListener('htmx:configRequest', function(evt) {
  const csrfToken = document.cookie.split('; ')
    .find(row => row.startsWith('csrf_token='))
    ?.split('=')[1];
  if (csrfToken) {
    evt.detail.headers['X-CSRF-Token'] = csrfToken;
  }
});
</script>
```

Ce script est **généré automatiquement** dans le template.

## Styling

### Inline Styles

```html
<style>
  .task-item { padding: 1rem; }
  .done { text-decoration: line-through; }
</style>
```

### External CSS

```html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/water.css@2/out/water.css">
```

### Tailwind CSS

```html
<script src="https://cdn.tailwindcss.com"></script>
<div class="p-4 bg-blue-500 text-white">Hello</div>
```

### Scoped Styles (via `<style scoped>`)

```gmx
<template>
<div class="component">Content</div>
</template>

<style scoped>
.component { color: red; }
</style>
```

!!!note "Limitation"
    Le scoping CSS n'est pas encore implémenté côté génération. `<style scoped>` génère actuellement du CSS global.

## Bonnes Pratiques

### ✅ Do

- Utiliser `{{route "..."}}` pour toutes les routes
- Valider les routes au compile-time
- Utiliser `hx-swap="outerHTML"` pour les mises à jour d'éléments
- Nommer les fragments de template (`{{define "task-item"}}`)
- Utiliser des IDs uniques (`id="task-{{.ID}}"`)

### ❌ Don't

- Ne pas hardcoder les chemins (`/api/tasks` ❌)
- Ne pas oublier `hx-target`
- Ne pas oublier `hx-swap`
- Ne pas ignorer les CSRF tokens
- Ne pas utiliser `innerHTML` sans échappement

## Limitations Actuelles

| Fonctionnalité | Status |
|----------------|--------|
| Go templates (if, range, etc.) | ✅ Implémenté |
| {{route "name"}} helper | ✅ Implémenté |
| HTMX attributes | ✅ Implémenté |
| CSRF auto-injection | ✅ Implémenté |
| Security headers | ✅ Implémenté |
| Template fragments | ✅ Implémenté |
| Scoped CSS | ❌ Non implémenté |
| Layouts / inheritance | ❌ Non implémenté |
| Partial rendering | ❌ Non implémenté |

## Prochaines Étapes

- **[Services](services.md)** — Configurer des services externes
- **[Security](security.md)** — Approfondir CSRF et validation
- **[Contributing](../contributing/generator.md)** — Voir comment les templates sont générés
