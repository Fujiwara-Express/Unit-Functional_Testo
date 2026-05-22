package middleware

import (
	"encoding/json"
	"net/http"
	"time"
)

// ErrorHandler is an error-handling middleware that recovers from panics
// and returns a standard HTTP 500 error response without exposing internal details.
func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := GetRequestID(r.Context())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":       "INTERNAL_SERVER_ERROR",
					"message":    "an internal server error occurred",
					"request_id": requestID,
					"timestamp":  time.Now().UTC().Format(time.RFC3339),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
