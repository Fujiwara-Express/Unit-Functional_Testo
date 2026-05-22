package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"routing-service/internal/apperrors"
	"routing-service/internal/clients"
	"routing-service/internal/middleware"
	"routing-service/internal/repositories"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func routerWithError(err error) *gin.Engine {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		_ = c.Error(err)
	})
	return r
}

func doGet(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestErrorHandler_ValidationError_Returns400(t *testing.T) {
	err := &apperrors.ValidationError{Message: "bad field", Fields: []string{"hub_id"}}
	w := doGet(routerWithError(err))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestErrorHandler_NotFoundError_Returns404(t *testing.T) {
	err := &apperrors.NotFoundError{Resource: "edge", ID: "e-1"}
	w := doGet(routerWithError(err))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestErrorHandler_RepoNotFound_Returns404(t *testing.T) {
	w := doGet(routerWithError(repositories.ErrNotFound))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestErrorHandler_DuplicateError_Returns409(t *testing.T) {
	err := &apperrors.DuplicateError{Resource: "hub", Key: "HUB_JKT"}
	w := doGet(routerWithError(err))
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestErrorHandler_RepoDuplicate_Returns409(t *testing.T) {
	w := doGet(routerWithError(repositories.ErrDuplicate))
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestErrorHandler_UpstreamUnavailable_Returns503(t *testing.T) {
	err := &apperrors.UpstreamUnavailableError{Service: "DeliveryService", Cause: errors.New("timeout")}
	w := doGet(routerWithError(err))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestErrorHandler_ClientUpstream_Returns503(t *testing.T) {
	err := &clients.ErrUpstreamUnavailable{Service: "DeliveryService", Cause: errors.New("timeout")}
	w := doGet(routerWithError(err))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestErrorHandler_UnknownError_Returns500(t *testing.T) {
	w := doGet(routerWithError(errors.New("something unexpected")))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestErrorHandler_NoError_PassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
