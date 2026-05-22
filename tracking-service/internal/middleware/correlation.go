package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

type contextKey string

// CorrelationIDKey is the context key used to store the correlation ID.
const CorrelationIDKey contextKey = "correlation_id"

// GetCorrelationID retrieves the correlation ID from the given context.
// Returns an empty string if no correlation ID is present.
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// generateUUID generates a random UUID v4.
func generateUUID() string {
	var uuid [16]byte
	rand.Read(uuid[:]) //nolint:errcheck // crypto/rand.Read never returns an error on supported platforms
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// CorrelationIDMiddleware reads the X-Correlation-ID request header. If the
// header is absent or empty a new UUID v4 is generated. The correlation ID is
// stored in the request context and echoed back in the X-Correlation-ID
// response header.
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = generateUUID()
		}
		ctx := context.WithValue(r.Context(), CorrelationIDKey, id)
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
