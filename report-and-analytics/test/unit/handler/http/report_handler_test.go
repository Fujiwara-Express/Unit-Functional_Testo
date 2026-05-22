package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/report-and-analytics/internal/domain"
	httphandler "github.com/report-and-analytics/internal/handler/http"
	"github.com/report-and-analytics/internal/handler/http/middleware"
	"github.com/report-and-analytics/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// injectRequestID injects a request_id into the request context (simulating middleware).
func injectRequestID(req *http.Request, id string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, id)
	return req.WithContext(ctx)
}

// ── GET /reports/orders ───────────────────────────────────────────────────────

func TestReportHandler_GetOrderReport_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetOrderReport(gomock.Any(), domain.OrderReportFilter{
		DateFrom: "2026-03-01",
		DateTo:   "2026-03-22",
		HubID:    "HUB_BDG",
	}).Return(&domain.OrderReport{
		Period:       "2026-03",
		TotalOrders:  150000,
		Delivered:    142000,
		Failed:       5000,
		Returned:     3000,
		SuccessRate:  94.67,
		TotalRevenue: 3750000000,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/orders?date_from=2026-03-01&date_to=2026-03-22&hub_id=HUB_BDG", nil)
	req = injectRequestID(req, "req-1")
	w := httptest.NewRecorder()

	handler.GetOrderReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp domain.OrderReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "2026-03", resp.Period)
	assert.Equal(t, int64(150000), resp.TotalOrders)
	assert.Equal(t, int64(142000), resp.Delivered)
	assert.Equal(t, 94.67, resp.SuccessRate)
	assert.Equal(t, int64(3750000000), resp.TotalRevenue)
}

func TestReportHandler_GetOrderReport_NoHubID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetOrderReport(gomock.Any(), domain.OrderReportFilter{
		DateFrom: "2026-03-01",
		DateTo:   "2026-03-31",
	}).Return(&domain.OrderReport{Period: "2026-03"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31", nil)
	req = injectRequestID(req, "req-1")
	w := httptest.NewRecorder()

	handler.GetOrderReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetOrderReport_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetOrderReport(gomock.Any(), gomock.Any()).Return(nil, domain.ErrValidation)

	req := httptest.NewRequest(http.MethodGet, "/reports/orders", nil)
	req = injectRequestID(req, "req-1")
	w := httptest.NewRecorder()

	handler.GetOrderReport(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	assert.NotEmpty(t, resp["request_id"])
}

func TestReportHandler_GetOrderReport_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetOrderReport(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31", nil)
	req = injectRequestID(req, "req-1")
	w := httptest.NewRecorder()

	handler.GetOrderReport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReportHandler_GetOrderReport_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetOrderReport(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31", nil)
	req = injectRequestID(req, "req-1")
	w := httptest.NewRecorder()

	handler.GetOrderReport(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INTERNAL_ERROR", resp["code"])
}

// ── GET /reports/delivery-performance ────────────────────────────────────────

func TestReportHandler_GetDeliveryPerformanceReport_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetDeliveryPerformanceReport(gomock.Any(), domain.DeliveryPerformanceFilter{
		CourierID: "CR123",
		Period:    "WEEKLY",
	}).Return(&domain.DeliveryPerformanceReport{
		CourierID:            "CR123",
		Period:               "WEEKLY",
		TotalJobs:            120,
		Delivered:            113,
		Failed:               5,
		Returned:             2,
		SuccessRate:          94.17,
		AvgDeliveryTimeHours: 3.5,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/delivery-performance?courier_id=CR123&period=WEEKLY", nil)
	req = injectRequestID(req, "req-2")
	w := httptest.NewRecorder()

	handler.GetDeliveryPerformanceReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp domain.DeliveryPerformanceReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "CR123", resp.CourierID)
	assert.Equal(t, "WEEKLY", resp.Period)
	assert.Equal(t, int64(120), resp.TotalJobs)
	assert.Equal(t, 94.17, resp.SuccessRate)
	assert.Equal(t, 3.5, resp.AvgDeliveryTimeHours)
}

func TestReportHandler_GetDeliveryPerformanceReport_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetDeliveryPerformanceReport(gomock.Any(), gomock.Any()).Return(nil, domain.ErrValidation)

	req := httptest.NewRequest(http.MethodGet, "/reports/delivery-performance?period=WEEKLY", nil)
	req = injectRequestID(req, "req-2")
	w := httptest.NewRecorder()

	handler.GetDeliveryPerformanceReport(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["request_id"])
}

func TestReportHandler_GetDeliveryPerformanceReport_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetDeliveryPerformanceReport(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/reports/delivery-performance?courier_id=CR123&period=WEEKLY", nil)
	req = injectRequestID(req, "req-2")
	w := httptest.NewRecorder()

	handler.GetDeliveryPerformanceReport(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GET /reports/revenue ──────────────────────────────────────────────────────

func TestReportHandler_GetRevenueReport_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetRevenueReport(gomock.Any(), domain.RevenueFilter{
		Period:      "MONTHLY",
		ServiceType: "REG",
	}).Return(&domain.RevenueReport{
		Period:        "MONTHLY",
		ServiceType:   "REG",
		TotalRevenue:  3750000000,
		TotalOrders:   150000,
		AvgOrderValue: 25000,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/revenue?period=MONTHLY&service_type=REG", nil)
	req = injectRequestID(req, "req-3")
	w := httptest.NewRecorder()

	handler.GetRevenueReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp domain.RevenueReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "MONTHLY", resp.Period)
	assert.Equal(t, "REG", resp.ServiceType)
	assert.Equal(t, int64(3750000000), resp.TotalRevenue)
	assert.Equal(t, float64(25000), resp.AvgOrderValue)
}

func TestReportHandler_GetRevenueReport_NoServiceType(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetRevenueReport(gomock.Any(), domain.RevenueFilter{
		Period: "MONTHLY",
	}).Return(&domain.RevenueReport{Period: "MONTHLY"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/revenue?period=MONTHLY", nil)
	req = injectRequestID(req, "req-3")
	w := httptest.NewRecorder()

	handler.GetRevenueReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetRevenueReport_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetRevenueReport(gomock.Any(), gomock.Any()).Return(nil, domain.ErrValidation)

	req := httptest.NewRequest(http.MethodGet, "/reports/revenue", nil)
	req = injectRequestID(req, "req-3")
	w := httptest.NewRecorder()

	handler.GetRevenueReport(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportHandler_GetRevenueReport_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetRevenueReport(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/reports/revenue?period=MONTHLY", nil)
	req = injectRequestID(req, "req-3")
	w := httptest.NewRecorder()

	handler.GetRevenueReport(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GET /reports/hub-performance ─────────────────────────────────────────────

func TestReportHandler_GetHubPerformanceReport_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetHubPerformanceReport(gomock.Any(), domain.HubPerformanceFilter{
		HubID:  "HUB_BDG",
		Period: "WEEKLY",
	}).Return(&domain.HubPerformanceReport{
		HubID:                  "HUB_BDG",
		Period:                 "WEEKLY",
		TotalInbound:           8500,
		TotalOutbound:          8200,
		TotalDispatched:        8000,
		AvgDwellTimeHours:      6.2,
		CapacityUtilizationPct: 72.5,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY", nil)
	req = injectRequestID(req, "req-4")
	w := httptest.NewRecorder()

	handler.GetHubPerformanceReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp domain.HubPerformanceReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "HUB_BDG", resp.HubID)
	assert.Equal(t, "WEEKLY", resp.Period)
	assert.Equal(t, int64(8500), resp.TotalInbound)
	assert.Equal(t, 6.2, resp.AvgDwellTimeHours)
	assert.Equal(t, 72.5, resp.CapacityUtilizationPct)
}

func TestReportHandler_GetHubPerformanceReport_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetHubPerformanceReport(gomock.Any(), gomock.Any()).Return(nil, domain.ErrValidation)

	req := httptest.NewRequest(http.MethodGet, "/reports/hub-performance?period=WEEKLY", nil)
	req = injectRequestID(req, "req-4")
	w := httptest.NewRecorder()

	handler.GetHubPerformanceReport(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["request_id"])
}

func TestReportHandler_GetHubPerformanceReport_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetHubPerformanceReport(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY", nil)
	req = injectRequestID(req, "req-4")
	w := httptest.NewRecorder()

	handler.GetHubPerformanceReport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReportHandler_GetHubPerformanceReport_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	handler := httphandler.NewReportHandler(mockSvc)

	mockSvc.EXPECT().GetHubPerformanceReport(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY", nil)
	req = injectRequestID(req, "req-4")
	w := httptest.NewRecorder()

	handler.GetHubPerformanceReport(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Auth middleware integration ───────────────────────────────────────────────

func TestReportHandler_Unauthorized_NoToken(t *testing.T) {
	endpoints := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"orders", "/reports/orders", nil},
		{"delivery-performance", "/reports/delivery-performance", nil},
		{"revenue", "/reports/revenue", nil},
		{"hub-performance", "/reports/hub-performance", nil},
	}

	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockReportService(ctrl)
	h := httphandler.NewReportHandler(mockSvc)

	handlers := []http.HandlerFunc{
		h.GetOrderReport,
		h.GetDeliveryPerformanceReport,
		h.GetRevenueReport,
		h.GetHubPerformanceReport,
	}

	for i, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			mux := http.NewServeMux()
			chain := func(next http.HandlerFunc) http.Handler {
				return middleware.RequestID(middleware.Auth(next))
			}
			mux.Handle("GET "+ep.path, chain(handlers[i]))

			req := httptest.NewRequest(http.MethodGet, ep.path, nil)
			// No Authorization header
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
