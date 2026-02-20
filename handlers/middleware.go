package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

// Response is the standard JSON envelope for all API responses.
type Response struct {
	Data  any    `json:"data"`
	Error string `json:"error,omitempty"`
}

// DB is the shared database connection used by all handlers.
var DB *sql.DB

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: msg})
}

// BasicAuth is middleware that enforces HTTP Basic Authentication.
func BasicAuth(next http.Handler) http.Handler {
	user := os.Getenv("AUTH_USER")
	pass := os.Getenv("AUTH_PASS")

	// If no credentials are configured, skip auth
	if user == "" && pass == "" {
		slog.Warn("AUTH_USER and AUTH_PASS not set, API is unauthenticated")
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="accounting"`)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
