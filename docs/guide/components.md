# Components

A GMX component is a single `.gmx` file that defines a complete feature: models, business logic, templates, and styles.

## File Structure

A `.gmx` file can contain four sections in **any order**:

```gmx
model Task { ... }          // Database models
service Database { ... }    // External services
<script>...</script>        // Business logic
<template>...</template>    // HTML + HTMX
<style scoped>...</style>   // CSS
```

All sections are optional, but most components will have at least models, script, and template.

## Complete Example

Here's a full-featured todo component (`examples/example.gmx`):

```gmx
service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}

service Mailer {
  provider: "smtp"
  host:     string @env("SMTP_HOST")
  pass:     string @env("SMTP_PASS")
  func send(to: string, subject: string, body: string) error
}

model Task {
  id:        uuid    @pk @default(uuid_v4)
  title:     string  @min(3) @max(255)
  done:      bool    @default(false)
  createdAt: datetime
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

func listTasks() error {
  let tasks = try Task.all()
  return render(tasks)
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
  <title>GMX Todo App</title>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
  <div class="container">
    <h1>📝 GMX Todo App</h1>

    <form class="task-form" hx-post="{{route "createTask"}}" hx-target="#task-list" hx-swap="beforeend">
      <input type="text" name="title" placeholder="What needs to be done?" required />
      <button type="submit">Add Task</button>
    </form>

    <ul id="task-list" class="task-list">
      {{range .Tasks}}
      <li class="task-item {{if .Done}}done{{end}}" id="task-{{.ID}}">
        <input
          type="checkbox"
          class="task-checkbox"
          {{if .Done}}checked{{end}}
          hx-patch="{{route "toggleTask"}}?id={{.ID}}"
          hx-target="#task-{{.ID}}"
          hx-swap="outerHTML"
        />
        <span class="task-title">{{.Title}}</span>
        <button
          class="task-delete"
          hx-delete="{{route "deleteTask"}}?id={{.ID}}"
          hx-target="#task-{{.ID}}"
          hx-swap="outerHTML swap:1s"
        >
          Delete
        </button>
      </li>
      {{end}}
    </ul>
  </div>
</body>
</html>
</template>

<style scoped>
  .task-item:hover {
    background: #f5f5f5;
  }
</style>
```

## Section Breakdown

### 1. Models

Define your database schema:

```gmx
model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3) @max(255)
  done:  bool   @default(false)
}
```

Generates GORM struct with validation. See [Models](models.md) for details.

### 2. Services

Configure external dependencies:

```gmx
service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}
```

Supports: `sqlite`, `postgres`, `smtp`, `http`. See [Services](services.md).

### 3. Script

Business logic in GMX Script (TypeScript-inspired):

```gmx
<script>
func createTask(title: string) error {
  const task = Task{title: title, done: false}
  try task.save()
  return render(task)
}
</script>
```

Transpiles to Go. See [Script](script.md).

### 4. Template

HTML with Go templates and HTMX:

```gmx
<template>
<button hx-post="{{route "createTask"}}">Create</button>
{{range .Tasks}}
  <div>{{.Title}}</div>
{{end}}
</template>
```

See [Templates](templates.md).

### 5. Style

Scoped or global CSS:

```gmx
<style scoped>
  .task-item { padding: 10px; }
</style>
```

`scoped` attribute isolates styles to this component.

## Section Order

Sections can appear in **any order**. These are equivalent:

```gmx
// Model-first
model Task { ... }
<script>...</script>
<template>...</template>
```

```gmx
// Template-first
<template>...</template>
<script>...</script>
model Task { ... }
```

**Best Practice:** Put models and services first for readability.

## Multiple Models

You can define multiple models in one file:

```gmx
model User {
  id:    uuid   @pk @default(uuid_v4)
  email: string @email @unique
}

model Post {
  id:     uuid   @pk @default(uuid_v4)
  userId: uuid   @relation(references: [User.id])
  title:  string
}
```

## Multiple Services

```gmx
service Database {
  provider: "postgres"
  url: string @env("DATABASE_URL")
}

service Mailer {
  provider: "smtp"
  host: string @env("SMTP_HOST")
  func send(to: string, subject: string, body: string) error
}
```

## Minimal Component

The absolute minimum is a template:

```gmx
<template>
<!DOCTYPE html>
<html>
<head><title>Hello</title></head>
<body><h1>Hello, GMX!</h1></body>
</html>
</template>
```

This generates a static page with no models or business logic.

## Data Flow

1. **Browser** sends HTTP request
2. **Handler** (generated) extracts parameters
3. **Script Function** (transpiled) executes business logic
4. **Model Methods** (generated) interact with database
5. **Template** (Go templates) renders response
6. **HTMX** swaps the HTML fragment

## Generated Code Structure

From a `.gmx` file, the compiler generates:

```go
package main

import ( /* auto-detected dependencies */ )

// ===== Models =====
type Task struct { ... }
func (t *Task) Validate() error { ... }

// ===== Services =====
type DatabaseConfig struct { ... }

// ===== Script (Transpiled) =====
func toggleTask(ctx *GMXContext, id string) error { ... }

// ===== Handlers =====
func handleToggleTask(w http.ResponseWriter, r *http.Request) { ... }

// ===== Template =====
var tmpl = template.Must(template.New("main").Parse(...))

// ===== Main =====
func main() {
  // Database init
  // Route registration
  // Server start
}
```

## Component Composition

For larger apps, split into multiple `.gmx` files:

```
components/
├── users.gmx       // User management
├── posts.gmx       // Blog posts
└── comments.gmx    // Comments on posts
```

Compile separately:

```bash
gmx components/users.gmx generated/users.go
gmx components/posts.gmx generated/posts.go
gmx components/comments.gmx generated/comments.go
```

Create a main file to combine them:

```go
package main

import (
  "generated/users"
  "generated/posts"
  "generated/comments"
)

func main() {
  mux := http.NewServeMux()
  users.RegisterRoutes(mux)
  posts.RegisterRoutes(mux)
  comments.RegisterRoutes(mux)

  http.ListenAndServe(":8080", mux)
}
```

## Best Practices

### ✅ Do

- Keep components focused on a single feature
- Use meaningful model and function names
- Add validation to models
- Handle errors in script functions
- Test generated code

### ❌ Don't

- Mix unrelated features in one component
- Skip error handling in script
- Ignore validation annotations
- Hardcode configuration (use services with `@env`)

## Next Steps

- **[Models](models.md)** — Learn about model definitions and annotations
- **[Script](script.md)** — Master GMX Script syntax
- **[Templates](templates.md)** — Build dynamic HTMX interfaces
