package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"tracking-service/internal/models"
)

// RecoveryMiddleware catches any panic that occurs in downstream handlers,
// logs the panic value and stack trace (with the correlation ID), and returns
// HTTP 500 with an ErrorResponse body.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				correlationID := GetCorrelationID(r.Context())
				stack := debug.Stack()

				fmt.Fprintf(os.Stderr, "panic recovered: %v correlation_id=%s\n%s\n",
					rec, correlationID, stack)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				resp := models.ErrorResponse{
					Error:         "internal server error",
					CorrelationID: correlationID,
				}
				json.NewEncoder(w).Encode(resp) //nolint:errcheck
			}
		}()

		next.ServeHTTP(w, r)
	})
}
