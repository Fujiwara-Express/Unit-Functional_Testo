package httphandler

// ErrorResponse represents a standard error response body.
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}
