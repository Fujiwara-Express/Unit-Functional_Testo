package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// Auth is a stub JWT authentication middleware.
// It checks for a Bearer token in the Authorization header.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":       "UNAUTHORIZED",
				"message":    "missing or invalid token",
				"request_id": GetRequestID(r.Context()),
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
