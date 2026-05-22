package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/fujiwara-express/warehouse-service/internal/domain"
	handler "github.com/fujiwara-express/warehouse-service/internal/delivery/http"
	"github.com/fujiwara-express/warehouse-service/mocks"
)

// ── HandleReceiveItem ────────────────────────────────────────────────────────

func TestHandleReceiveItem_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	item := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}

	mockSvc.EXPECT().
		ReceiveItem(gomock.Any(), item).
		Return(nil).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	body, _ := json.Marshal(item)
	req := httptest.NewRequest(http.MethodPost, "/receive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleReceiveItem(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "berhasil masuk")
}

func TestHandleReceiveItem_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().ReceiveItem(gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodPost, "/receive", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()

	h.HandleReceiveItem(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleReceiveItem_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		ReceiveItem(gomock.Any(), gomock.Any()).
		Return(assert.AnError).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	body, _ := json.Marshal(domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 5})
	req := httptest.NewRequest(http.MethodPost, "/receive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleReceiveItem(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleReceiveItem_WrongMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().ReceiveItem(gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/receive", nil)
	rec := httptest.NewRecorder()

	h.HandleReceiveItem(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// ── HandleDispatchItem ───────────────────────────────────────────────────────

func TestHandleDispatchItem_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		DispatchItem(gomock.Any(), "BRG-001", 3).
		Return(nil).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	body, _ := json.Marshal(domain.StockRequest{ItemID: "BRG-001", Quantity: 3})
	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleDispatchItem(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "berhasil dikeluarkan")
}

func TestHandleDispatchItem_OutOfStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		DispatchItem(gomock.Any(), "BRG-001", 999).
		Return(domain.ErrOutOfStock).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	body, _ := json.Marshal(domain.StockRequest{ItemID: "BRG-001", Quantity: 999})
	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleDispatchItem(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleDispatchItem_ItemNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		DispatchItem(gomock.Any(), "BRG-999", 1).
		Return(domain.ErrItemNotFound).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	body, _ := json.Marshal(domain.StockRequest{ItemID: "BRG-999", Quantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleDispatchItem(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleDispatchItem_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().DispatchItem(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()

	h.HandleDispatchItem(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleDispatchItem_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		DispatchItem(gomock.Any(), "BRG-001", 1).
		Return(assert.AnError).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	body, _ := json.Marshal(domain.StockRequest{ItemID: "BRG-001", Quantity: 1})
	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.HandleDispatchItem(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleDispatchItem_WrongMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().DispatchItem(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/dispatch", nil)
	rec := httptest.NewRecorder()

	h.HandleDispatchItem(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// ── HandleCheckStock ─────────────────────────────────────────────────────────

func TestHandleCheckStock_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	expected := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}

	mockSvc.EXPECT().
		CheckStock(gomock.Any(), "BRG-001").
		Return(expected, nil).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/check-stock?id=BRG-001", nil)
	rec := httptest.NewRecorder()

	h.HandleCheckStock(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got domain.Item
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, expected.ID, got.ID)
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.Quantity, got.Quantity)
	assert.Equal(t, expected.Location, got.Location)
}

func TestHandleCheckStock_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		CheckStock(gomock.Any(), "BRG-999").
		Return(domain.Item{}, domain.ErrItemNotFound).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/check-stock?id=BRG-999", nil)
	rec := httptest.NewRecorder()

	h.HandleCheckStock(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleCheckStock_MissingIDParam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().CheckStock(gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/check-stock", nil)
	rec := httptest.NewRecorder()

	h.HandleCheckStock(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCheckStock_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().
		CheckStock(gomock.Any(), "BRG-001").
		Return(domain.Item{}, assert.AnError).
		Times(1)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/check-stock?id=BRG-001", nil)
	rec := httptest.NewRecorder()

	h.HandleCheckStock(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCheckStock_WrongMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWarehouseService(ctrl)
	mockSvc.EXPECT().CheckStock(gomock.Any(), gomock.Any()).Times(0)

	h := handler.NewWarehouseHandler(mockSvc)
	req := httptest.NewRequest(http.MethodPost, "/check-stock?id=BRG-001", nil)
	rec := httptest.NewRecorder()

	h.HandleCheckStock(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
