# GMX Compiler — Roadmap to Production

## Current State
PoC with basic lexer/parser/generator pipeline. Generates a working Go binary from a simple .gmx file but with major gaps vs spec v1.4.

## Phase 1: Foundation Reset
**Goal**: Clean codebase, establish proper .gmx file format, fix all deprecations.

- [ ] Replace `ioutil.ReadFile` with `os.ReadFile`
- [ ] Replace `strings.Title` with `cases.Title` from `golang.org/x/text`
- [ ] Define canonical .gmx file format with section delimiters: `--- model ---`, `--- script ---`, `--- template ---`, `--- style ---`
- [ ] Create `examples/todo.gmx` reference file matching the spec v1.4 (Task model with all annotations)
- [ ] Add `internal/compiler/errors/errors.go` with position-aware error types (file, line, col, message)

## Phase 2: New Lexer
**Goal**: Production-grade tokenizer with full GMX language support.

- [ ] Rune-based lexer (unicode support)
- [ ] Line/column tracking on every token
- [ ] Comment support (`//` line comments, `/* */` block comments)
- [ ] All operators: `==`, `!=`, `<=`, `>=`, `&&`, `||`, `?`, `:`
- [ ] Proper semicolon handling (or newline-as-statement-terminator like Go)
- [ ] String interpolation tokens: detect `"Tâche: {t.title}"` as a sequence of STRING_PART + EXPR + STRING_PART
- [ ] Section delimiter tokens: `--- model ---` etc.
- [ ] All GMX keywords: `model`, `service`, `func`, `task`, `let`, `try`, `if`, `else`, `return`, `import`, `true`, `false`
- [ ] Annotation tokens: `@pk`, `@default(...)`, `@min(...)`, `@max(...)`, `@email`, `@unique`, `@scoped`, `@relation(...)`, `@env(...)`, `@async`, `@cron(...)`
- [ ] Comprehensive test suite

## Phase 3: Section-Aware Parser
**Goal**: Split .gmx files into typed sections, parse each with specialized sub-parsers.

- [ ] Top-level parser that splits on section delimiters
- [ ] Model section parser: full field declarations with types + annotations + annotation arguments
- [ ] Script section parser: function declarations with typed args, return types, body as statement list
- [ ] Template section: stored as raw HTML string for Phase 6
- [ ] Style section: stored as raw CSS string for Phase 8
- [ ] Support for `services.gmx` files (model-less, service-only)
- [ ] New AST nodes for all constructs
- [ ] Comprehensive test suite

## Phase 4: Model System (Code Generation)
**Goal**: Generate production-quality Go structs, DB layer, and validation from model declarations.

- [ ] Go struct generation with proper field types (uuid→string, bool, int, string, float, datetime)
- [ ] GORM tags: primaryKey, unique, default values, column names
- [ ] Foreign key generation from @relation
- [ ] Array/slice fields for reverse relations (Post[] → []Post)
- [ ] Validation functions generated from @min, @max, @email constraints
- [ ] @default values in GORM tags
- [ ] @scoped field tracking (stored in metadata for Phase 10)
- [ ] SQL migration file generation (CREATE TABLE DDL)
- [ ] Generated ORM methods: Model.Find(id), Model.All(), Model.Create(), Model.Update(), Model.Delete()
- [ ] Test: generated code compiles with `go build`

## Phase 5: GMX Script Transpiler
**Goal**: Transpile GMX Script (TypeScript-like) to valid Go code.

- [ ] Statement parser: let, try, if/else, return, assignments, expressions
- [ ] Expression parser (Pratt parser): binary ops, unary ops, method calls, field access, function calls
- [ ] `let x = expr` → `x := expr`
- [ ] `try expr` → `result, err := expr; if err != nil { return err }`
- [ ] `Task.find(id)` → `db.First(&task, "id = ?", id)` (using generated ORM)
- [ ] `task.save()` → `db.Save(&task)`
- [ ] `render(task)` → template execution returning HTML fragment
- [ ] String interpolation `"Tâche: {t.title}"` → `fmt.Sprintf("Tâche: %s", t.Title)`
- [ ] Type inference from model definitions
- [ ] Function signature transpilation (typed args → Go args)
- [ ] Inject `ctx` parameter (carries tenant, auth, request)
- [ ] Test: generated Go functions compile and run

## Phase 6: Template Engine
**Goal**: Compile GMX templates to Go html/template with type-safe helpers.

- [ ] Parse `<template>...</template>` content
- [ ] Expression interpolation `{expr}` → `{{.Expr}}` or custom template func
- [ ] Ternary expressions `{x ? "a" : "b"}` → template helper
- [ ] Function calls in attributes `hx-patch={toggleTask(Task.id)}` → URL generation
- [ ] Loop support (if keeping JSX-like maps or switching to range)
- [ ] Component-scoped template names
- [ ] Template compilation to `html/template` with auto-escaping (XSS protection)
- [ ] Test: templates render correct HTML

## Phase 7: HTMX Route Resolution
**Goal**: Auto-generate HTTP handlers from function references in templates.

- [ ] Scan templates for `hx-get`, `hx-post`, `hx-put`, `hx-patch`, `hx-delete` attributes
- [ ] Map function references to HTTP routes: `toggleTask(Task.id)` → `PATCH /api/task/:id/toggle`
- [ ] Generate route registration code
- [ ] Generate handler wrappers that: parse URL params, call the transpiled function, return HTML fragment
- [ ] Type-safe URL helper functions for templates
- [ ] OOB swap support: `render(Task, SidebarCounter)` → concatenated HTML with `hx-swap-oob="true"`
- [ ] CSRF token middleware (auto-inject in forms + HTMX headers)
- [ ] Test: HTTP handlers respond correctly

## Phase 8: Service Infrastructure
**Goal**: Full services.gmx support with dependency injection.

- [ ] Parse `service Name { provider, fields, funcs }` blocks
- [ ] Database service: provider "postgres" or "sqlite", connection setup
- [ ] Mailer service: provider "smtp", generated send function
- [ ] Storage service: provider "s3", file ops
- [ ] @env injection → os.Getenv with validation
- [ ] Service constructors with proper error handling
- [ ] Dependency injection into handler context
- [ ] Middleware chain generation (auth, tenant, logging)
- [ ] Test: services initialize and inject correctly

## Phase 9: Style System
**Goal**: Scoped CSS with optional Tailwind support.

- [ ] Parse `<style>...</style>` blocks
- [ ] Component-scoped CSS (prefix classes with component hash)
- [ ] Embed CSS in generated binary via go:embed
- [ ] Tailwind CDN support (default) or JIT build pipeline
- [ ] CSS injection in HTML head

## Phase 10: Advanced Features
**Goal**: Multi-tenancy, background tasks, imports.

- [ ] @scoped: inject `WHERE tenant_id = ?` on all generated ORM queries
- [ ] Tenant extraction from context (middleware)
- [ ] Background tasks: `task Name @async { ... }` → goroutine wrapper
- [ ] Cron tasks: `task Name @cron("...") { ... }` → cron scheduler integration
- [ ] Import system: `import "github.com/pkg" as Alias` → go.mod management
- [ ] StdLib GMX: pre-injected Crypto, String, Date, Log, JSON helpers
- [ ] Native bridges: `provider: "native"` pointing to .go files

## Phase 11: Build Pipeline & CLI
**Goal**: Complete `gmx build` and `gmx dev` commands.

- [ ] `gmx build` command: parse → transpile → generate → go build → single binary
- [ ] `go:embed` for static assets (CSS, templates if needed)
- [ ] `gmx dev` command: file watcher + auto-rebuild + live reload
- [ ] `gmx init` command: scaffold new project
- [ ] Generated go.mod/go.sum management
- [ ] Output directory structure (.gmx-out/)
- [ ] Error reporting with source mapping (GMX line → Go line)

## Phase 12: Integration Tests & Example App
**Goal**: Prove the compiler works end-to-end with a real app.

- [ ] Complete Todo app example (CRUD + multi-tenant)
- [ ] Blog app example (relations, OOB swaps)
- [ ] End-to-end test: .gmx → compile → run → HTTP assertions
- [ ] Benchmark: compilation time, binary size, request latency
- [ ] README.md with getting started guide
