package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the HTTP status code
// written by the handler.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before delegating to the underlying
// ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// newResponseWriter creates a responseWriter with a default status of 200.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// NewLoggingMiddleware returns a logging middleware that writes one structured
// log line per request to the provided io.Writer. The log line uses key=value
// format and includes: method, path, status, latency_ms, and correlation_id.
func NewLoggingMiddleware(out io.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			latencyMs := time.Since(start).Milliseconds()
			correlationID := GetCorrelationID(r.Context())

			fmt.Fprintf(out, "method=%s path=%s status=%d latency_ms=%d correlation_id=%s\n",
				r.Method,
				r.URL.Path,
				rw.statusCode,
				latencyMs,
				correlationID,
			)
		})
	}
}

// LoggingMiddleware is the default logging middleware that writes to os.Stderr.
var LoggingMiddleware = NewLoggingMiddleware(os.Stderr)
