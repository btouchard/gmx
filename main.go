package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
)

// ========== Helper Functions ==========

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

// ========== Main ==========

func main() {
	databaseCfg := initDatabase()

	_ = databaseCfg

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)

	fmt.Println("GMX server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", csrfProtect(securityHeaders(mux))))
}
