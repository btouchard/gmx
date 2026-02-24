<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/HTMX-3366CC?style=for-the-badge&logo=htmx&logoColor=white" alt="HTMX" />
  <img src="https://img.shields.io/badge/Single_Binary-black?style=for-the-badge" alt="Single Binary" />
</p>

<h1 align="center">GMX</h1>
<p align="center"><strong>The full-stack Go framework that thinks in components.</strong></p>
<p align="center">Write Vue-style single-file components. Ship a single Go binary.<br/>No Node. No JS bundler. No runtime. Just Go + HTMX.</p>

<p align="center">
  <a href="#quickstart">Quickstart</a> •
  <a href="#why-gmx">Why GMX</a> •
  <a href="#the-gmx-file">The .gmx File</a> •
  <a href="#features">Features</a> •
  <a href="#documentation">Docs</a> •
  <a href="#roadmap">Roadmap</a>
</p>

---

## What is GMX?

GMX is a **transpiler framework** that compiles `.gmx` single-file components into production-ready **Go** applications with **HTMX** interactivity.

One file. Models, logic, templates, styles — all colocated. One command. A single, dependency-free binary.

```
todo.gmx  →  gmx build  →  ./todo (single binary, ~5MB, serves on :8080)
```

GMX doesn't hide Go or HTMX. It **makes them work together** with type safety, auto-generated routes, built-in security, and zero JavaScript.

---

## Quickstart

```bash
# Install
go install github.com/kolapsis/gmx/cmd/gmx@latest

# Create your first component
cat > todo.gmx << 'EOF'
<script>
model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3) @max(255)
  done:  bool   @default(false)
}

service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}

func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}
</script>

<template>
  <ul>
    {{range .Tasks}}
    <li id="task-{{.ID}}">
      <button hx-patch="{{route `toggleTask`}}?id={{.ID}}"
              hx-target="#task-{{.ID}}" hx-swap="outerHTML">
        {{if .Done}}✓{{else}}○{{end}} {{.Title}}
      </button>
    </li>
    {{end}}
  </ul>
</template>

<style>
  li { padding: 0.5rem; cursor: pointer; }
  .done { text-decoration: line-through; opacity: 0.5; }
</style>
EOF

# Build & run
DATABASE_URL="app.db" gmx build todo.gmx
./main
```

That's it. You have a working CRUD app with HTMX reactivity, SQLite persistence, input validation, and CSRF protection. In **one file**.

---

## Why GMX?

### The Problem

Building modern web apps means choosing between two extremes:

| | **JS Frameworks** (Next, Nuxt, SvelteKit) | **Go Frameworks** (Gin, Echo, Fiber) |
|---|---|---|
| DX | Great (components, hot reload) | Verbose (scattered files) |
| Performance | Runtime overhead, hydration | Fast, but no component model |
| Deployment | Node runtime, Docker images | Single binary ✓ |
| Type safety | Partial (runtime errors) | Strong ✓ |
| Bundle | JS + CSS + sourcemaps | Just a binary |

### The GMX Answer

GMX gives you **Vue's developer experience** with **Go's production characteristics**:

- **Component colocation** — Model, logic, template, style in one file
- **Single binary output** — No runtime, no Docker, just `scp` and run
- **Type-safe HTMX** — Auto-generated routes, validated parameters, no broken links
- **Zero JavaScript** — HTMX handles interactivity, Go handles everything else
- **Built-in security** — CSRF, XSS escaping, SQL injection prevention, all automatic

---

## The `.gmx` File

A `.gmx` file is a **single-file component** inspired by Vue's SFC format. Everything your feature needs lives in one place.

```html
<script>
// ── Imports ──────────────────────────────────────
import TaskItem from "./components/TaskItem.gmx"
import { sendEmail } from "./services/mailer.gmx"
import "github.com/stripe/stripe-go" as Stripe

// ── Constants & Variables ────────────────────────
const MAX_TASKS = 100
let requestCount: int = 0

// ── Models (auto-generates DB schema + ORM) ─────
model Task {
  id:        uuid     @pk @default(uuid_v4)
  title:     string   @min(3) @max(255)
  done:      bool     @default(false)
  priority:  int      @min(1) @max(5) @default(3)
  tenantId:  uuid     @scoped          // ← auto multi-tenancy
  author:    User     @relation(references: [id])
}

// ── Services (infra config, 12-factor) ──────────
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

// ── Handlers (auto-routed, type-checked) ────────
func createTask(title: string, priority: int) error {
  if title == "" {
    return error("Title cannot be empty")
  }
  const task = Task{title: title, priority: priority, done: false}
  try task.save()
  return render(task)
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

<template>
  <form hx-post="{{route `createTask`}}" hx-target="#task-list" hx-swap="beforeend">
    <input name="title" placeholder="What needs to be done?" required />
    <button type="submit">Add</button>
  </form>

  <ul id="task-list">
    {{range .Tasks}}
    <li id="task-{{.ID}}" class="{{if .Done}}done{{end}}">
      <button hx-patch="{{route `toggleTask`}}?id={{.ID}}"
              hx-target="#task-{{.ID}}" hx-swap="outerHTML">
        {{if .Done}}✓{{else}}○{{end}} {{.Title}}
      </button>
      <button hx-delete="{{route `deleteTask`}}?id={{.ID}}"
              hx-target="#task-{{.ID}}" hx-swap="outerHTML"
              hx-confirm="Delete this task?">×</button>
    </li>
    {{end}}
  </ul>
</template>

<style>
  form { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
  li { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem; }
  .done { text-decoration: line-through; opacity: 0.5; }
</style>
```

### What the compiler does with this

| You write | GMX generates |
|-----------|---------------|
| `model Task { ... }` | Go struct + GORM tags + validation + ORM methods + SQL migrations |
| `func toggleTask(...)` | HTTP handler + route registration + param parsing + CSRF check |
| `{{route `toggleTask`}}` | Type-safe URL `/api/toggleTask` (compile error if function doesn't exist) |
| `@scoped` on `tenantId` | `WHERE tenant_id = ?` injected on every query, automatically |
| `@min(3) @max(255)` | Server-side validation before any DB operation |
| `<style>` | Scoped CSS embedded in the binary via `go:embed` |
| `import X from "Y.gmx"` | Recursive multi-file resolution, AST merging, template composition |

---

## Features

### 🧩 Component System
- **Single-file components** with `<script>`, `<template>`, `<style>` sections
- **Import system** — Vue-style default, destructured, and Go native imports
- **Multi-file compilation** with recursive dependency resolution and circular import detection
- **Scoped CSS** with automatic class prefixing

### 🗄️ Data Layer
- **Declarative models** with type-safe annotations (`@pk`, `@unique`, `@email`, `@min`, `@max`, `@default`, `@relation`)
- **Auto-generated ORM** — `Task.find(id)`, `Task.all()`, `.save()`, `.delete()`
- **Multi-tenancy** — `@scoped` injects tenant isolation on all queries
- **Database providers** — SQLite & PostgreSQL via service configuration

### ⚡ HTMX Integration
- **Typed route resolution** — `{{route "funcName"}}` validated at compile time
- **Auto handler generation** — functions become HTTP endpoints with correct methods
- **OOB swaps** — `render(Task, SidebarCounter)` for multi-target updates
- **Fragment rendering** — handlers return HTML partials, not full pages

### 🔒 Security (Built-in, not Bolt-on)
- **CSRF protection** — Double-submit cookies, auto-injected in forms and HTMX headers
- **XSS prevention** — Contextual auto-escaping via Go's `html/template`
- **SQL injection** — Parameterized queries only, no string concatenation
- **Input validation** — Model constraints enforced server-side before every operation
- **UUID validation** — Path parameters validated before reaching handlers
- **Security headers** — Middleware with CSP, X-Frame-Options, etc.

### 🏗️ Infrastructure
- **Services** — Database, SMTP, HTTP clients, S3 storage as typed declarations
- **Environment config** — `@env("VAR")` with validation, 12-factor compliant
- **Dependency injection** — Services auto-injected into handler context
- **Go imports** — `import "github.com/pkg" as Alias` maps directly to `go.mod`

### 📦 Build & Deploy
- **Single binary** — `gmx build` → one file, no runtime dependencies
- **Embedded assets** — CSS, templates compiled in via `go:embed`
- **~5MB binaries** — Go's static compilation, nothing extra
- **Zero Docker needed** — `scp binary server:/ && ./binary`

---

## Import System

GMX supports three import styles, all inside `<script>`:

```javascript
// 1. Component import (like Vue)
// Imports the component's template, models, and styles
import TaskItem from "./components/TaskItem.gmx"

// 2. Destructured import (pick what you need)
// Cherry-pick functions, models, or services from another file
import { sendEmail, MailerConfig } from "./services/mailer.gmx"

// 3. Go native import (use any Go package)
// Adds to go.mod, available with alias in your script
import "github.com/stripe/stripe-go" as Stripe
```

Imports are **resolved recursively** — if `TaskItem.gmx` imports `Badge.gmx`, it just works. Circular imports are detected at compile time.

---

## GMX Script

GMX Script is a **TypeScript-inspired syntax** that transpiles to Go. It's intentionally small — not a new language, but a **thin layer** over Go with better ergonomics for web handlers.

| GMX Script | Generated Go |
|-----------|-------------|
| `let task = try Task.find(id)` | `task, err := TaskFind(db, id); if err != nil { return err }` |
| `try task.save()` | `if err := TaskSave(db, task); if err != nil { return err }` |
| `return render(task)` | `return tmpl.ExecuteTemplate(w, "task", task)` |
| `return error("Not found")` | `return fmt.Errorf("Not found")` |
| `let userId = ctx.User` | `userId := ctx.User` |
| `"Task: {t.title}"` | `fmt.Sprintf("Task: %s", t.Title)` |

Error handling uses `try` (unwrap-or-return), inspired by Rust/Swift. No more `if err != nil` boilerplate.

---

## Architecture

```
  .gmx file
      │
      ▼
┌──────────┐    ┌──────────┐    ┌───────────┐    ┌───────────┐    ┌──────────┐
│  Lexer   │ →  │  Parser  │ →  │ Resolver  │ →  │ Generator │ →  │ go build │
│ (tokens) │    │  (AST)   │    │ (imports) │    │ (Go code) │    │ (binary) │
└──────────┘    └──────────┘    └───────────┘    └───────────┘    └──────────┘
                     │                                  │
                     ▼                                  ▼
              Script Parser                    gen_models.go
              (GMX Script → AST)               gen_handlers.go
                                               gen_template.go
                                               gen_imports.go
                                               gen_services.go
                                               gen_vars.go
                                               gen_helpers.go
                                               gen_main.go
                                               gen_components.go
```

The compiler is fully **modular** — each phase is independently testable with **91%+ test coverage**.

---

## Comparison

| | **GMX** | **Templ + HTMX** | **Next.js** | **Laravel** |
|---|:---:|:---:|:---:|:---:|
| Single-file components | ✅ | ❌ | ✅ | ✅ (Blade) |
| Type-safe routes | ✅ | ❌ | ❌ | ❌ |
| Single binary | ✅ | ✅ | ❌ | ❌ |
| Zero JS needed | ✅ | ✅ | ❌ | ✅ (optional) |
| Auto multi-tenancy | ✅ | ❌ | ❌ | ❌ |
| Built-in CSRF | ✅ | Manual | ✅ | ✅ |
| Auto ORM from schema | ✅ | ❌ | Prisma | Eloquent |
| Component imports | ✅ | ❌ | ✅ | ✅ |
| No runtime deps | ✅ | ✅ | ❌ | ❌ |
| Learning curve | Low | Medium | High | Medium |

---

## Project Structure

```
your-app/
├── app.gmx                    # Main component (entry point)
├── components/
│   ├── TaskItem.gmx           # Reusable component
│   └── Navbar.gmx
├── services/
│   └── mailer.gmx             # Shared service + functions
└── .env                       # Environment variables
```

```bash
gmx build app.gmx              # → produces ./main binary
DATABASE_URL="app.db" ./main    # → serves on :8080
```

---

## Documentation

Full documentation available at [gmx.dev](https://gmx.dev) (coming soon) or locally:

```bash
pip install mkdocs-material
mkdocs serve
# → http://127.0.0.1:8000
```

**Guides**: Getting Started, Components, Models, Script, Templates, Services, Security

**Contributing**: Architecture, AST Reference, Lexer & Parser, Generator, Script Transpiler, Testing

---

## Roadmap

- [x] Lexer with unicode, line/col tracking, all operators
- [x] Section-aware parser (model, service, func, let/const, import)
- [x] GMX Script transpiler (let, try, if/else, render, error, ctx)
- [x] Code generator (models, handlers, templates, routes, main)
- [x] Service infrastructure (SQLite, PostgreSQL, SMTP, HTTP)
- [x] Security (CSRF, XSS, SQL injection, UUID validation, headers)
- [x] Import system (Vue-style, destructured, Go native)
- [x] Multi-file compilation with recursive resolution
- [x] Scoped CSS
- [ ] `gmx dev` — File watcher + live reload
- [ ] Background tasks (`@async`, `@cron`)
- [ ] OOB swap generation (`render(A, B)` → concatenated HTML)
- [ ] Tailwind JIT integration
- [ ] `gmx init` — Project scaffolding
- [ ] Source maps (GMX line → Go line)

---

## Contributing

GMX is open source and contributions are welcome.

```bash
git clone https://github.com/kolapsis/gmx.git
cd gmx
go test ./...                  # Run all tests (~91% coverage)
go build -o gmx ./cmd/gmx     # Build the compiler
```

The codebase is structured for clarity: `internal/compiler/` contains the lexer, parser, resolver, script transpiler, and generator — each with comprehensive tests.

See [CONTRIBUTING.md](docs/contributing/) for architecture details.

---

## License

Apache 2.0 — see [LICENSE](LICENSE)

The code **generated** by GMX belongs entirely to you. The Apache 2.0 license applies only to the GMX compiler itself.

---

<p align="center">
  <strong>Stop shipping JavaScript. Start shipping binaries.</strong><br/>
  <a href="#quickstart">Get started →</a>
</p>
