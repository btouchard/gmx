package main

import (
	"crypto/rand"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
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

// isValidEmail checks if a string looks like a valid email
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// scopedDB returns a DB handle filtered by tenant_id for multi-tenant isolation
func scopedDB(db *gorm.DB, tenantID string) *gorm.DB {
	return db.Where("tenant_id = ?", tenantID)
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

// generateCSRFToken generates a cryptographically secure random token
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// csrfProtect is a middleware that provides CSRF protection using double-submit cookie pattern
func csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe methods: pass through, ensure cookie exists
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			// Set CSRF cookie if not present
			if _, err := r.Cookie("_csrf"); err != nil {
				token := generateCSRFToken()
				http.SetCookie(w, &http.Cookie{
					Name:     "_csrf",
					Value:    token,
					Path:     "/",
					HttpOnly: false, // JS needs to read it for HTMX header
					SameSite: http.SameSiteStrictMode,
					Secure:   r.TLS != nil,
				})
			}
			next.ServeHTTP(w, r)
			return
		}

		// Mutating methods: validate CSRF token
		cookie, err := r.Cookie("_csrf")
		if err != nil {
			http.Error(w, "Forbidden - missing CSRF cookie", http.StatusForbidden)
			return
		}

		// Check header first (HTMX sends this), then form value (regular forms)
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			token = r.FormValue("_csrf")
		}

		if token == "" || token != cookie.Value {
			http.Error(w, "Forbidden - invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
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

// ========== Variables ==========

const APP_NAME = "GMX Demo"
const MAX_TITLE_LENGTH = 255

var requestCount int = 0
var debugMode bool = false

// ========== Models ==========

type User struct {
	ID        string    `gorm:"primaryKey;default:uuid_v4" json:"id"`
	Email     string    `gorm:"unique" json:"email"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Score     float64   `json:"score"`
	IsAdmin   bool      `gorm:"default:false" json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
	Tasks     []Task    `json:"tasks"`
}

// Validate checks all field constraints defined in the model
func (u *User) Validate() error {
	if u.Email != "" && !isValidEmail(u.Email) {
		return fmt.Errorf("email: invalid email format")
	}
	if len(u.Name) < 2 {
		return fmt.Errorf("name: minimum length is 2, got %d", len(u.Name))
	}
	if len(u.Name) > 100 {
		return fmt.Errorf("name: maximum length is 100, got %d", len(u.Name))
	}
	if u.Age < 18 {
		return fmt.Errorf("age: minimum value is 18, got %v", u.Age)
	}
	if u.Age > 150 {
		return fmt.Errorf("age: maximum value is 150, got %v", u.Age)
	}
	if u.Score < 0 {
		return fmt.Errorf("score: minimum value is 0, got %v", u.Score)
	}
	return nil
}

// BeforeCreate is a GORM hook that sets default values before inserting
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = generateUUID()
	}
	return nil
}

type Task struct {
	ID        string    `gorm:"primaryKey;default:uuid_v4" json:"id"`
	Title     string    `json:"title"`
	Done      bool      `gorm:"default:false" json:"done"`
	Priority  int       `gorm:"default:3" json:"priority"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
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
	if t.Priority < 1 {
		return fmt.Errorf("priority: minimum value is 1, got %v", t.Priority)
	}
	if t.Priority > 5 {
		return fmt.Errorf("priority: maximum value is 5, got %v", t.Priority)
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

// GitHubConfig holds configuration for the GitHub service
type GitHubConfig struct {
	Provider string
	BaseUrl  string
	ApiKey   string
}

// initGitHub initializes the GitHub service configuration
func initGitHub() *GitHubConfig {
	cfg := &GitHubConfig{
		Provider: "http",
	}
	cfg.BaseUrl = os.Getenv("GITHUB_API_URL")
	if cfg.BaseUrl == "" {
		log.Fatal("missing required env var: GITHUB_API_URL")
	}
	cfg.ApiKey = os.Getenv("GITHUB_TOKEN")
	if cfg.ApiKey == "" {
		log.Fatal("missing required env var: GITHUB_TOKEN")
	}
	return cfg
}

// GitHubClient is an HTTP client for the GitHub service
type GitHubClient struct {
	config *GitHubConfig
	http   *http.Client
}

// newGitHubClient creates a new HTTP client for GitHub
func newGitHubClient(cfg *GitHubConfig) *GitHubClient {
	return &GitHubClient{
		config: cfg,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Get makes a GET request to the API
func (c *GitHubClient) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.config.BaseUrl+path, nil)
	if err != nil {
		return nil, err
	}
	if c.config.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
	}
	return c.http.Do(req)
}

// Post makes a POST request to the API
func (c *GitHubClient) Post(path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.config.BaseUrl+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
	}
	return c.http.Do(req)
}

// ========== Script (Transpiled) ==========

// ORM helper functions

func UserFind(db *gorm.DB, id string) (*User, error) {
	var obj User
	if err := db.First(&obj, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

func UserAll(db *gorm.DB) ([]User, error) {
	var objs []User
	if err := db.Find(&objs).Error; err != nil {
		return nil, err
	}
	return objs, nil
}

func UserSave(db *gorm.DB, obj *User) error {
	return db.Save(obj).Error
}

func UserDelete(db *gorm.DB, obj *User) error {
	return db.Delete(obj).Error
}

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

// gmx:69
func listTasks(ctx *GMXContext) error {
	// gmx:70
	tasks, err := TaskAll(ctx.DB)
	if err != nil {
		return err
	}
	// gmx:71
	for _, item := range tasks {
		if err := renderFragment(ctx.Writer, "Task", item); err != nil {
			return err
		}
	}
	return nil
}

// gmx:75
func getTask(ctx *GMXContext, id string) error {
	// gmx:76
	task, err := TaskFind(ctx.DB, id)
	if err != nil {
		return err
	}
	// gmx:77
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// gmx:81
func createTask(ctx *GMXContext, title string, priority int) error {
	// gmx:82
	if title == "" {
		// gmx:83
		return fmt.Errorf("Title cannot be empty")
	}
	// gmx:86
	task := &Task{Title: title, Priority: priority, Done: false, UserID: ctx.User}
	// gmx:93
	if err := TaskSave(ctx.DB, task); err != nil {
		return err
	}
	// gmx:94
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// gmx:98
func toggleTask(ctx *GMXContext, id string) error {
	// gmx:99
	task, err := TaskFind(ctx.DB, id)
	if err != nil {
		return err
	}
	// gmx:100
	task.Done = !task.Done
	// gmx:101
	if err := TaskSave(ctx.DB, task); err != nil {
		return err
	}
	// gmx:102
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// gmx:106
func updateTask(ctx *GMXContext, id string, title string) error {
	// gmx:107
	task, err := TaskFind(ctx.DB, id)
	if err != nil {
		return err
	}
	// gmx:108
	task.Title = title
	// gmx:109
	if err := TaskSave(ctx.DB, task); err != nil {
		return err
	}
	// gmx:110
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// gmx:114
func deleteTask(ctx *GMXContext, id string) error {
	// gmx:115
	task, err := TaskFind(ctx.DB, id)
	if err != nil {
		return err
	}
	// gmx:116
	if err := TaskDelete(ctx.DB, task); err != nil {
		return err
	}
	// gmx:117
	return nil
}

// gmx:121
func getTaskLabel(ctx *GMXContext, t *Task) string {
	// gmx:122
	label := fmt.Sprintf("Task: %v", t.Title)
	// gmx:123
	return label
}

// gmx:127
func createUserTask(ctx *GMXContext, title string) error {
	// gmx:128
	userId := ctx.User
	// gmx:129
	tenantId := ctx.Tenant
	// gmx:131
	if userId == "" {
		// gmx:132
		return fmt.Errorf("User not authenticated")
	}
	// gmx:135
	task := &Task{TenantID: tenantId, Title: title, Done: false, UserID: userId}
	// gmx:142
	if err := TaskSave(ctx.DB, task); err != nil {
		return err
	}
	// gmx:143
	if err := renderFragment(ctx.Writer, "Task", task); err != nil {
		return err
	}
	return nil
}

// ========== Script Handler Wrappers ==========

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

func handleGetTask(w http.ResponseWriter, r *http.Request) {
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

	if err := getTask(ctx, id); err != nil {
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
	// Extract parameter: priority
	priority := r.PathValue("priority")
	if priority == "" {
		priority = r.FormValue("priority")
	}
	if priority == "" {
		http.Error(w, "Missing required parameter: priority", http.StatusBadRequest)
		return
	}
	priorityInt, err := strconv.Atoi(priority)
	if err != nil {
		http.Error(w, "Invalid integer parameter", http.StatusBadRequest)
		return
	}

	if err := createTask(ctx, title, priorityInt); err != nil {
		log.Printf("handler error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

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

func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
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
	// Extract parameter: title
	title := r.PathValue("title")
	if title == "" {
		title = r.FormValue("title")
	}
	if title == "" {
		http.Error(w, "Missing required parameter: title", http.StatusBadRequest)
		return
	}

	if err := updateTask(ctx, id, title); err != nil {
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

func handleCreateUserTask(w http.ResponseWriter, r *http.Request) {
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

	if err := createUserTask(ctx, title); err != nil {
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
				"createTask": "/api/createTask",
				"listTasks":  "/api/listTasks",
				"toggleTask": "/api/toggleTask",
				"deleteTask": "/api/deleteTask",
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
  <title>GMX Demo — Full Feature Showcase</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <style>
  /* GMX Scoped Styles */
  .task-item:focus-within {
    outline: 2px solid #007bff;
    outline-offset: -2px;
    border-radius: 4px;
  }

  </style>
  <meta name="csrf-token" content="{{.CSRFToken}}">
  <script>
    document.addEventListener('DOMContentLoaded', function() {
      document.body.addEventListener('htmx:configRequest', function(e) {
        var token = document.querySelector('meta[name="csrf-token"]');
        if (token) {
          e.detail.headers['X-CSRF-Token'] = token.content;
        }
      });
    });
  </script>
</head>
<body>
  <div class="container">
    <header>
      <h1>GMX Demo</h1>
      <p>Tasks: {{len .Tasks}}</p>
    </header>

    <!-- CSRF token is auto-available as {{.CSRFToken}} -->

    <!-- POST: create a task -->
    <form class="task-form"
      hx-post="{{route "createTask"}}"
      hx-target="#task-list"
      hx-swap="beforeend">
      <input type="text" name="title" placeholder="What needs to be done?" required />
      <select name="priority">
        <option value="1">Low</option>
        <option value="3" selected>Normal</option>
        <option value="5">High</option>
      </select>
      <button type="submit">Add Task</button>
    </form>

    <!-- GET: load all tasks -->
    <section
      hx-get="{{route "listTasks"}}"
      hx-trigger="load"
      hx-target="#task-list"
      hx-swap="innerHTML">
    </section>

    <ul id="task-list" class="task-list">
      {{range .Tasks}}{{template "Task" .}}{{end}}
    </ul>

    {{if eq (len .Tasks) 0}}
      <p class="empty-state">No tasks yet. Add one above!</p>
    {{end}}
  </div>
</body>
</html>
<!-- ========== Model Fragment Templates ========== -->

{{define "Task"}}
      <li class="task-item {{if .Done}}done{{end}}" id="task-{{.ID}}">

        <!-- PATCH: toggle task -->
        <input
          type="checkbox"
          class="task-checkbox"
          {{if .Done}}checked{{end}}
          hx-patch="{{route "toggleTask"}}?id={{.ID}}"
          hx-target="#task-{{.ID}}"
          hx-swap="outerHTML"
        />

        <span class="task-title">{{.Title}}</span>

        {{if gt .Priority 3}}
          <span class="badge priority-high">High</span>
        {{else if eq .Priority 3}}
          <span class="badge priority-normal">Normal</span>
        {{else}}
          <span class="badge priority-low">Low</span>
        {{end}}

        {{if not .Done}}
          <span class="badge pending">Pending</span>
        {{end}}

        <!-- DELETE: remove task -->
        <button
          class="task-delete"
          hx-delete="{{route "deleteTask"}}?id={{.ID}}"
          hx-target="#task-{{.ID}}"
          hx-swap="outerHTML swap:300ms"
          hx-confirm="Delete this task?"
        >
          Delete
        </button>
      </li>
      {{end}}
`

// ========== Page Data ==========

type PageData struct {
	CSRFToken string
	Users     []User
	Tasks     []Task
}

// ========== Database ==========

var db *gorm.DB

// ========== Handlers ==========

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// Get or create CSRF token
	csrfToken := ""
	if cookie, err := r.Cookie("_csrf"); err == nil {
		csrfToken = cookie.Value
	} else {
		csrfToken = generateCSRFToken()
		http.SetCookie(w, &http.Cookie{
			Name:     "_csrf",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})
	}

	data := PageData{
		CSRFToken: csrfToken,
		// Fetch User from database
		Users: []User{},
		// Fetch Task from database
		Tasks: []Task{},
	}

	// Fetch data from database
	db.Find(&data.Users)
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
	gitHubCfg := initGitHub()
	gitHubClient := newGitHubClient(gitHubCfg)

	_ = mailerCfg
	_ = mailerSvc
	_ = gitHubCfg
	_ = gitHubClient

	var err error
	db, err = gorm.Open(sqlite.Open(databaseCfg.Url), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	db.AutoMigrate(&User{}, &Task{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/updateTask", handleUpdateTask)
	mux.HandleFunc("/api/createUserTask", handleCreateUserTask)
	mux.HandleFunc("/api/createTask", handleCreateTask)
	mux.HandleFunc("/api/listTasks", handleListTasks)
	mux.HandleFunc("/api/toggleTask", handleToggleTask)
	mux.HandleFunc("/api/deleteTask", handleDeleteTask)
	mux.HandleFunc("/api/getTask", handleGetTask)

	fmt.Println("GMX server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", csrfProtect(securityHeaders(mux))))
}
