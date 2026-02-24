package main

import (
	"crypto/rand"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"
)

// ========== Helper Functions ==========

// generateUUID generates a UUID v4 string
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// isValidUUID checks if a string is a valid UUID v4 format
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// securityHeaders adds security headers to HTTP responses
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// ========== Models ==========

type Task struct {
	ID        string    `gorm:"primaryKey;default:uuid_v4" json:"id"`
	Title     string    `json:"title"`
	Done      bool      `gorm:"default:false" json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

// Validate checks all field constraints defined in the model
func (t *Task) Validate() error {
	if len(t.Title) < 3 {
		return fmt.Errorf("title: minimum length is 3, got %d", len(t.Title))
	}
	if len(t.Title) > 255 {
		return fmt.Errorf("title: maximum length is 255, got %d", len(t.Title))
	}
	return nil
}

// BeforeCreate is a GORM hook that sets default values before inserting
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = generateUUID()
	}
	return nil
}

// ========== Services ==========

// DatabaseConfig holds configuration for the Database service
type DatabaseConfig struct {
	Provider string
	Url      string
}

// initDatabase initializes the Database service configuration
func initDatabase() *DatabaseConfig {
	cfg := &DatabaseConfig{
		Provider: "sqlite",
	}
	cfg.Url = os.Getenv("DATABASE_URL")
	if cfg.Url == "" {
		log.Fatal("missing required env var: DATABASE_URL")
	}
	return cfg
}

// MailerConfig holds configuration for the Mailer service
type MailerConfig struct {
	Provider string
	Host     string
	Pass     string
}

// initMailer initializes the Mailer service configuration
func initMailer() *MailerConfig {
	cfg := &MailerConfig{
		Provider: "smtp",
	}
	cfg.Host = os.Getenv("SMTP_HOST")
	if cfg.Host == "" {
		log.Fatal("missing required env var: SMTP_HOST")
	}
	cfg.Pass = os.Getenv("SMTP_PASS")
	if cfg.Pass == "" {
		log.Fatal("missing required env var: SMTP_PASS")
	}
	return cfg
}

// MailerService defines the interface for the Mailer service
type MailerService interface {
	Send(to string, subject string, body string) error
}

// mailerImpl is an SMTP implementation of MailerService
type mailerImpl struct {
	config *MailerConfig
}

func (m *mailerImpl) Send(to string, subject string, body string) error {
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"\r\n" +
		body)

	var auth smtp.Auth
	if m.config.Pass != "" {
		auth = smtp.PlainAuth("", "", m.config.Pass, m.config.Host)
	}

	from := "noreply@localhost"
	addr := m.config.Host
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

// newMailerService creates a new SMTP instance of MailerService
func newMailerService(cfg *MailerConfig) MailerService {
	return &mailerImpl{config: cfg}
}

// ========== Script (Transpiled) ==========

// ORM helper functions

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
	return db.Save(obj).Error
}

func TaskDelete(db *gorm.DB, obj *Task) error {
	return db.Delete(obj).Error
}

// GMXContext holds request context and dependencies
type GMXContext struct {
	DB      *gorm.DB
	Tenant  string
	User    string
	Writer  http.ResponseWriter
	Request *http.Request
}

// renderFragment executes a template fragment
func renderFragment(w http.ResponseWriter, name string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, name, data)
}

// gmx:21
func toggleTask(ctx *GMXContext, id string) error {
	// gmx:22
	task, err := TaskFind(ctx.DB, id)
	if err != nil {
		return err
	}
	// gmx:23
	task.Done = !task.Done
	// gmx:24
	if err := TaskSave(ctx.DB, task); err != nil {
		return err
	}
	// gmx:25
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// gmx:28
func createTask(ctx *GMXContext, title string) error {
	// gmx:29
	if title == "" {
		// gmx:30
		return fmt.Errorf("Title cannot be empty")
	}
	// gmx:33
	task := &Task{Title: title, Done: false}
	// gmx:34
	if err := TaskSave(ctx.DB, task); err != nil {
		return err
	}
	// gmx:35
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// gmx:38
func listTasks(ctx *GMXContext) error {
	// gmx:39
	tasks, err := TaskAll(ctx.DB)
	if err != nil {
		return err
	}
	// gmx:40
	if err := renderFragment(ctx.Writer, "Task", tasks); err != nil {
		return err
	}
	return nil
}

// gmx:43
func deleteTask(ctx *GMXContext, id string) error {
	// gmx:44
	task, err := TaskFind(ctx.DB, id)
	if err != nil {
		return err
	}
	// gmx:45
	if err := TaskDelete(ctx.DB, task); err != nil {
		return err
	}
	// gmx:46
	return nil
}

// ========== Script Handler Wrappers ==========

func handleToggleTask(w http.ResponseWriter, r *http.Request) {
	// Method guard
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := &GMXContext{
		DB:      db,
		Writer:  w,
		Request: r,
	}

	// Extract parameter: id
	id := r.PathValue("id")
	if id == "" {
		id = r.FormValue("id")
	}
	if id == "" {
		http.Error(w, "Missing required parameter: id", http.StatusBadRequest)
		return
	}
	if !isValidUUID(id) {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	if err := toggleTask(ctx, id); err != nil {
		log.Printf("handler error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	// Method guard
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := &GMXContext{
		DB:      db,
		Writer:  w,
		Request: r,
	}

	// Extract parameter: title
	title := r.PathValue("title")
	if title == "" {
		title = r.FormValue("title")
	}
	if title == "" {
		http.Error(w, "Missing required parameter: title", http.StatusBadRequest)
		return
	}

	if err := createTask(ctx, title); err != nil {
		log.Printf("handler error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	// Method guard
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := &GMXContext{
		DB:      db,
		Writer:  w,
		Request: r,
	}

	if err := listTasks(ctx); err != nil {
		log.Printf("handler error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	// Method guard
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := &GMXContext{
		DB:      db,
		Writer:  w,
		Request: r,
	}

	// Extract parameter: id
	id := r.PathValue("id")
	if id == "" {
		id = r.FormValue("id")
	}
	if id == "" {
		http.Error(w, "Missing required parameter: id", http.StatusBadRequest)
		return
	}
	if !isValidUUID(id) {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	if err := deleteTask(ctx, id); err != nil {
		log.Printf("handler error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// ========== Template ==========

var tmpl *template.Template

func init() {
	funcMap := template.FuncMap{
		"route": func(name string) string {
			routes := map[string]string{
				"deleteTask": "/api/deleteTask",
				"createTask": "/api/createTask",
				"toggleTask": "/api/toggleTask",
			}
			if r, ok := routes[name]; ok {
				return r
			}
			return "/api/" + name
		},
	}

	tmpl = template.Must(template.New("page").Funcs(funcMap).Parse(pageTemplate))
}

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GMX Todo App</title>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      max-width: 800px;
      margin: 0 auto;
      padding: 2rem;
      background: #f5f5f5;
    }
    .container {
      background: white;
      padding: 2rem;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    h1 {
      color: #333;
      margin-bottom: 2rem;
    }
    .task-form {
      display: flex;
      gap: 1rem;
      margin-bottom: 2rem;
    }
    .task-form input {
      flex: 1;
      padding: 0.75rem;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 1rem;
    }
    .task-form button {
      padding: 0.75rem 1.5rem;
      background: #007bff;
      color: white;
      border: none;
      border-radius: 4px;
      font-size: 1rem;
      cursor: pointer;
    }
    .task-form button:hover {
      background: #0056b3;
    }
    .task-list {
      list-style: none;
      padding: 0;
    }
    .task-item {
      display: flex;
      align-items: center;
      padding: 1rem;
      border-bottom: 1px solid #eee;
      transition: background 0.2s;
    }
    .task-item:hover {
      background: #f9f9f9;
    }
    .task-item.done .task-title {
      text-decoration: line-through;
      opacity: 0.6;
    }
    .task-checkbox {
      margin-right: 1rem;
      width: 20px;
      height: 20px;
      cursor: pointer;
    }
    .task-title {
      flex: 1;
    }
    .task-delete {
      padding: 0.5rem 1rem;
      background: #dc3545;
      color: white;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      font-size: 0.875rem;
    }
    .task-delete:hover {
      background: #c82333;
    }
  </style>
  <style>
  /* GMX Scoped Styles */
  .task-item:hover {
    background: #f5f5f5;
  }
  </style>
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
</html>`

// ========== Page Data ==========

type PageData struct {
	Tasks []Task
}

// ========== Database ==========

var db *gorm.DB

// ========== Handlers ==========

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		// Fetch Task from database
		Tasks: []Task{},
	}

	// Fetch data from database
	db.Find(&data.Tasks)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ========== Main ==========

func main() {
	databaseCfg := initDatabase()
	mailerCfg := initMailer()
	mailerSvc := newMailerService(mailerCfg)

	_ = mailerCfg
	_ = mailerSvc

	var err error
	db, err = gorm.Open(sqlite.Open(databaseCfg.Url), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	db.AutoMigrate(&Task{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/createTask", handleCreateTask)
	mux.HandleFunc("/api/toggleTask", handleToggleTask)
	mux.HandleFunc("/api/deleteTask", handleDeleteTask)
	mux.HandleFunc("/api/listTasks", handleListTasks)

	fmt.Println("GMX server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", securityHeaders(mux)))
}
