# Getting Started

This guide will walk you through installing GMX and building your first application.

## Prerequisites

- **Go 1.21 or later** — [Download Go](https://go.dev/dl/)
- **Basic understanding of HTML and Go** (helpful but not required)

## Installation

### Option 1: Build from Source (Current)

```bash
git clone https://github.com/btouchard/gmx.git
cd gmx
go build -o gmx cmd/gmx/main.go
sudo mv gmx /usr/local/bin/
```

### Option 2: Go Install (Coming Soon)

```bash
go install github.com/btouchard/gmx/cmd/gmx@latest
```

Verify installation:

```bash
gmx
# Output: Usage: gmx <input.gmx>
```

## Your First GMX App

Create a file named `hello.gmx`:

```gmx
<script>
service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}

model Message {
  id:      uuid    @pk @default(uuid_v4)
  content: string  @min(1) @max(255)
}

func listMessages() error {
  let messages = try Message.all()
  return render(messages)
}

func createMessage(content: string) error {
  if content == "" {
    return error("Content cannot be empty")
  }

  const msg = Message{content: content}
  try msg.save()
  return render(msg)
}
</script>

<template>
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Hello GMX</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</head>
<body>
  <h1>Messages</h1>

  <form hx-post="{{route "createMessage"}}" hx-target="#messages" hx-swap="beforeend">
    <input type="text" name="content" placeholder="Type a message" required />
    <button type="submit">Send</button>
  </form>

  <div id="messages">
    {{range .Messages}}
    <div>{{.Content}}</div>
    {{end}}
  </div>
</body>
</html>
</template>
```

## Compile Your App

```bash
gmx hello.gmx main.go
```

This generates a complete Go web server in `main.go` (approximately 200-300 lines of code).

## Run Your App

```bash
go run main.go
```

Open your browser to `http://localhost:8080` and you'll see your message app running!

## What Just Happened?

The GMX compiler transformed your `.gmx` file into:

1. **GORM Models** with validation and UUID generation
2. **HTTP Handlers** with method guards and parameter validation
3. **Business Logic** transpiled from GMX Script to Go
4. **Template Setup** with HTMX route helpers
5. **Main Function** with database initialization and routing
6. **Security Middleware** with CSRF protection

All in a single, compilable Go file.

## Project Structure

For a real application, organize your code like this:

```
myapp/
├── components/
│   ├── tasks.gmx
│   ├── users.gmx
│   └── dashboard.gmx
├── generated/
│   └── *.go
├── main.go
└── go.mod
```

Compile all components:

```bash
for file in components/*.gmx; do
  gmx "$file" "generated/$(basename "$file" .gmx).go"
done
```

## Next Steps

- **[Components](components.md)** — Learn about the structure of `.gmx` files
- **[Models](models.md)** — Deep dive into model declarations and annotations
- **[Script](script.md)** — GMX Script language reference
- **[Templates](templates.md)** — Master HTMX templates and routing

## Common Issues

### "Command not found: gmx"

Make sure `/usr/local/bin` is in your PATH:

```bash
echo $PATH
# Should include /usr/local/bin
```

### Generated Code Doesn't Compile

The GMX compiler runs `gofmt` on generated code. If compilation fails:

1. Check for syntax errors in your `.gmx` file
2. Ensure all GMX Script functions return `error`
3. Verify model field types are valid (`uuid`, `string`, `int`, `bool`, `datetime`)

### Database Errors

GMX generates code that expects a SQLite database. Make sure:

```bash
go get gorm.io/gorm
go get gorm.io/driver/sqlite
```

## Getting Help

- **Issues:** [github.com/btouchard/gmx/issues](https://github.com/btouchard/gmx/issues)
- **Discussions:** [github.com/btouchard/gmx/discussions](https://github.com/btouchard/gmx/discussions)
