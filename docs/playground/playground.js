// Import CodeMirror 6 from CDN
// EditorState is re-exported by codemirror — importing it from the same package
// ensures a single @codemirror/state instance across all extensions.
import { EditorView, basicSetup } from 'https://esm.sh/codemirror@6.0.1';
import { EditorState } from 'https://esm.sh/@codemirror/state@^6.0.0';
import { oneDark } from 'https://esm.sh/@codemirror/theme-one-dark@6.1.2';
import { go } from 'https://esm.sh/@codemirror/lang-go@6.0.1';
import { javascript } from 'https://esm.sh/@codemirror/lang-javascript@6.2.2';

// Import GMX language mode
import { gmxLanguage } from './gmx-lang.js';

// Examples
const EXAMPLES = {
  'minimal': `<script>
model User {
  id:    uuid   @pk @default(uuid_v4)
  name:  string @min(2) @max(100)
  email: string @email @unique
}

func getUser(id: uuid) error {
  let user = try User.find(id)
  return render(user)
}
</script>

<template>
<!DOCTYPE html>
<html>
<head><title>User Profile</title></head>
<body>
  <h1>{{.User.Name}}</h1>
  <p>{{.User.Email}}</p>
</body>
</html>
</template>`,

  'task-app': `<script>
// Full-featured task management app

const APP_NAME = "Task Manager"
let requestCount: int = 0

service Database {
  provider: "sqlite"
  url:      string @env("DATABASE_URL")
}

model Task {
  id:        uuid     @pk @default(uuid_v4)
  title:     string   @min(3) @max(255)
  done:      bool     @default(false)
  priority:  int      @min(1) @max(5) @default(3)
  createdAt: datetime
}

// GET handler
func listTasks() error {
  let tasks = try Task.all()
  return render(tasks)
}

// POST handler
func createTask(title: string, priority: int) error {
  if title == "" {
    return error("Title cannot be empty")
  }

  const task = Task{
    title: title,
    priority: priority,
    done: false
  }

  try task.save()
  return render(task)
}

// PATCH handler
func toggleTask(id: uuid) error {
  let task = try Task.find(id)
  task.done = !task.done
  try task.save()
  return render(task)
}

// DELETE handler
func deleteTask(id: uuid) error {
  let task = try Task.find(id)
  try task.delete()
  return nil
}
</script>

<template>
<!DOCTYPE html>
<html>
<head>
  <title>Task Manager</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</head>
<body>
  <h1>Tasks ({{len .Tasks}})</h1>

  <form hx-post="{{route "createTask"}}" hx-target="#task-list">
    <input type="text" name="title" placeholder="New task..." required />
    <select name="priority">
      <option value="1">Low</option>
      <option value="3" selected>Normal</option>
      <option value="5">High</option>
    </select>
    <button type="submit">Add</button>
  </form>

  <ul id="task-list">
    {{range .Tasks}}
    <li id="task-{{.ID}}">
      <input type="checkbox" {{if .Done}}checked{{end}}
        hx-patch="{{route "toggleTask"}}?id={{.ID}}"
        hx-target="#task-{{.ID}}" />
      <span>{{.Title}}</span>
      <button hx-delete="{{route "deleteTask"}}?id={{.ID}}"
        hx-target="#task-{{.ID}}">Delete</button>
    </li>
    {{end}}
  </ul>
</body>
</html>
</template>`,

  'services': `<script>
// Service declarations

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

service GitHub {
  provider: "http"
  baseUrl:  string @env("GITHUB_API_URL")
  apiKey:   string @env("GITHUB_TOKEN")
}

model User {
  id:    uuid   @pk @default(uuid_v4)
  email: string @email @unique
}

func notifyUser(userId: uuid) error {
  let user = try User.find(userId)
  try Mailer.send(user.email, "Welcome!", "Hello from GMX")
  return render(user)
}
</script>

<template>
<!DOCTYPE html>
<html>
<head><title>Services Demo</title></head>
<body>
  <h1>Email sent to {{.User.Email}}</h1>
</body>
</html>
</template>`
};

// Global state
let gmxEditor, goEditor;
let wasmReady = false;
let compileTimeout = null;

// Initialize WASM
async function initWASM() {
  const go = new Go();

  try {
    const result = await WebAssembly.instantiateStreaming(
      fetch('gmx.wasm'),
      go.importObject
    );

    go.run(result.instance);
    wasmReady = true;
    hideLoading();

    // Initial compilation
    compile();
  } catch (err) {
    showError(['Failed to load WASM runtime: ' + err.message]);
    hideLoading();
  }
}

// Initialize editors
function initEditors() {
  // GMX editor (left pane)
  gmxEditor = new EditorView({
    state: EditorState.create({
      doc: EXAMPLES['minimal'],
      extensions: [
        basicSetup,
        oneDark,
        gmxLanguage(),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && document.getElementById('auto-compile').checked) {
            scheduleCompile();
          }
        })
      ]
    }),
    parent: document.getElementById('gmx-editor')
  });

  // Go editor (right pane, read-only)
  goEditor = new EditorView({
    state: EditorState.create({
      doc: '// Compiled Go code will appear here...',
      extensions: [
        basicSetup,
        oneDark,
        go(),
        EditorState.readOnly.of(true)
      ]
    }),
    parent: document.getElementById('go-editor')
  });
}

// Compile GMX source
function compile() {
  if (!wasmReady) {
    showError(['WASM runtime not ready']);
    return;
  }

  const source = gmxEditor.state.doc.toString();
  const startTime = performance.now();

  try {
    const result = window.compileGMX(source);
    const elapsed = (performance.now() - startTime).toFixed(0);

    if (result.errors && result.errors.length > 0) {
      showError(result.errors);
    } else {
      hideError();
    }

    // Update Go editor
    goEditor.dispatch({
      changes: {
        from: 0,
        to: goEditor.state.doc.length,
        insert: result.code || '// No output'
      }
    });

    // Show compilation time
    document.getElementById('compile-time').textContent = `${elapsed}ms`;
  } catch (err) {
    showError(['Compilation failed: ' + err.message]);
  }
}

// Schedule compilation with debounce
function scheduleCompile() {
  clearTimeout(compileTimeout);
  compileTimeout = setTimeout(compile, 300);
}

// Show/hide loading overlay
function showLoading() {
  document.getElementById('loading').classList.remove('hidden');
}

function hideLoading() {
  document.getElementById('loading').classList.add('hidden');
}

// Show/hide error panel
function showError(errors) {
  const panel = document.getElementById('error-panel');
  const content = document.getElementById('error-content');

  // Clear previous errors
  content.textContent = '';

  // Add each error as a separate div using safe DOM methods
  errors.forEach(err => {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-item';
    errorDiv.textContent = err;
    content.appendChild(errorDiv);
  });

  panel.classList.remove('hidden');
}

function hideError() {
  document.getElementById('error-panel').classList.add('hidden');
}

// Handle example selection
function loadExample(exampleId) {
  if (!EXAMPLES[exampleId]) return;

  gmxEditor.dispatch({
    changes: {
      from: 0,
      to: gmxEditor.state.doc.length,
      insert: EXAMPLES[exampleId]
    }
  });

  if (document.getElementById('auto-compile').checked) {
    compile();
  }
}

// Handle split-pane resizing
function initResizer() {
  const resizer = document.getElementById('resizer');
  const leftPane = document.querySelector('.pane-left');
  const rightPane = document.querySelector('.pane-right');

  let isResizing = false;

  resizer.addEventListener('mousedown', (e) => {
    isResizing = true;
    document.body.style.cursor = 'col-resize';
    e.preventDefault();
  });

  document.addEventListener('mousemove', (e) => {
    if (!isResizing) return;

    const container = document.querySelector('.panes');
    const containerRect = container.getBoundingClientRect();
    const offsetX = e.clientX - containerRect.left;
    const percentage = (offsetX / containerRect.width) * 100;

    if (percentage > 20 && percentage < 80) {
      leftPane.style.width = `${percentage}%`;
      rightPane.style.width = `${100 - percentage}%`;
    }
  });

  document.addEventListener('mouseup', () => {
    isResizing = false;
    document.body.style.cursor = '';
  });
}

// Event listeners
document.getElementById('compile-btn').addEventListener('click', compile);
document.getElementById('close-errors').addEventListener('click', hideError);
document.getElementById('example-selector').addEventListener('change', (e) => {
  loadExample(e.target.value);
});

// Initialize
showLoading();
initEditors();
initResizer();
initWASM();
