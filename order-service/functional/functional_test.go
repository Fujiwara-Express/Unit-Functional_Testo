// Package functional contains functional tests for the order-service.
//
// Unlike unit tests (which mock every dependency), functional tests wire the
// real Handler and real Service together and use an in-memory fake repository.
// This exercises the full HTTP request → handler → service → repository path
// without requiring a live database, verifying that all three layers integrate
// correctly end-to-end.
package functional

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/handler"
	"order-service/service"
	"order-service/types"
)

// ─── In-memory fake repository ────────────────────────────────────────────────

// fakeRepo is a thread-safe in-memory implementation of types.OrderRepository.
// It is used in functional tests to avoid a real database while still exercising
// the real service and handler logic.
type fakeRepo struct {
	mu     sync.RWMutex
	orders map[string]*types.Order
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{orders: make(map[string]*types.Order)}
}

func (f *fakeRepo) SaveOrder(_ context.Context, order *types.Order) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *order
	f.orders[order.OrderID] = &cp
	return nil
}

func (f *fakeRepo) FindOrderByID(_ context.Context, orderID string) (*types.Order, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	o, ok := f.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order %s: %w", orderID, types.ErrNotFound)
	}
	cp := *o
	return &cp, nil
}

func (f *fakeRepo) FindOrders(_ context.Context, params types.ListOrdersParams) ([]*types.Order, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var result []*types.Order
	for _, o := range f.orders {
		if params.UserID != "" && o.SenderUserID != params.UserID {
			continue
		}
		if params.Status != "" && o.Status != params.Status {
			continue
		}
		cp := *o
		result = append(result, &cp)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * limit
	if start >= len(result) {
		return []*types.Order{}, nil
	}
	end := start + limit
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (f *fakeRepo) UpdateOrder(_ context.Context, order *types.Order) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.orders[order.OrderID]; !ok {
		return fmt.Errorf("order %s: %w", order.OrderID, types.ErrNotFound)
	}
	cp := *order
	f.orders[order.OrderID] = &cp
	return nil
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

// testEnv holds a wired-up handler + router for functional tests.
type testEnv struct {
	repo    *fakeRepo
	handler *handler.Handler
	mux     *http.ServeMux
}

// newTestEnv creates a fresh environment for each test.
// It uses Go 1.22 ServeMux method+path patterns so that r.PathValue is
// populated automatically for wildcard segments like {order_id}.
func newTestEnv() *testEnv {
	repo := newFakeRepo()
	svc := service.NewOrderService(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.CreateOrder)
	mux.HandleFunc("GET /orders", h.ListOrders)
	mux.HandleFunc("GET /orders/{order_id}", h.GetOrder)
	mux.HandleFunc("POST /orders/{order_id}/cancel", h.CancelOrder)
	mux.HandleFunc("PATCH /orders/{order_id}", h.UpdateOrder)

	return &testEnv{repo: repo, handler: h, mux: mux}
}

// do sends a request through the mux and returns the recorded response.
func (e *testEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)
	return w
}

// validCreateBody returns a minimal valid create-order request body.
func validCreateBody() map[string]any {
	return map[string]any{
		"sender_user_id":   "user-001",
		"sender_name":      "Alice",
		"sender_address":   "123 Main St",
		"sender_phone":     "08111111111",
		"sender_city_code": "JKT",
		"receiver_name":    "Bob",
		"receiver_address": "456 Oak Ave",
		"receiver_phone":   "08222222222",
		"weight":           2.5,
		"service_type":     "REG",
		"item_description": "Books",
	}
}

// ─── Create Order functional tests ────────────────────────────────────────────

func TestFunctional_CreateOrder_HappyPath(t *testing.T) {
	env := newTestEnv()
	w := env.do("POST", "/orders", validCreateBody())

	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp["order_id"], "order_id must be present")
	assert.NotEmpty(t, resp["tracking_number"], "tracking_number must be present")
	assert.Greater(t, resp["price"].(float64), 0.0, "price must be > 0")
	assert.NotEmpty(t, resp["status"], "status must be present")
	assert.NotEmpty(t, resp["created_at"], "created_at must be present")

	// Verify the order was actually persisted in the repo
	orderID := resp["order_id"].(string)
	stored, err := env.repo.FindOrderByID(context.Background(), orderID)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusAwaitingPickup, stored.Status)
	assert.Equal(t, "user-001", stored.SenderUserID)
}

func TestFunctional_CreateOrder_MalformedJSON(t *testing.T) {
	env := newTestEnv()
	req := httptest.NewRequest("POST", "/orders", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFunctional_CreateOrder_MissingRequiredField(t *testing.T) {
	env := newTestEnv()
	body := validCreateBody()
	delete(body, "sender_user_id")
	w := env.do("POST", "/orders", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFunctional_CreateOrder_InvalidServiceType(t *testing.T) {
	env := newTestEnv()
	body := validCreateBody()
	body["service_type"] = "INVALID"
	w := env.do("POST", "/orders", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFunctional_CreateOrder_CODWithZeroAmount(t *testing.T) {
	env := newTestEnv()
	body := validCreateBody()
	body["is_cod"] = true
	body["cod_amount"] = 0
	w := env.do("POST", "/orders", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFunctional_CreateOrder_AllServiceTypes(t *testing.T) {
	serviceTypes := []string{"REG", "YES", "OKE", "SAME_DAY"}
	for _, st := range serviceTypes {
		t.Run(st, func(t *testing.T) {
			env := newTestEnv()
			body := validCreateBody()
			body["service_type"] = st
			w := env.do("POST", "/orders", body)

			require.Equal(t, http.StatusCreated, w.Code)
			var resp map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Greater(t, resp["price"].(float64), 0.0)
		})
	}
}

func TestFunctional_CreateOrder_InsuranceSurcharge(t *testing.T) {
	body := validCreateBody()

	envNoIns := newTestEnv()
	body["insurance"] = false
	wNoIns := envNoIns.do("POST", "/orders", body)
	require.Equal(t, http.StatusCreated, wNoIns.Code)
	var respNoIns map[string]any
	require.NoError(t, json.Unmarshal(wNoIns.Body.Bytes(), &respNoIns))

	envIns := newTestEnv()
	body["insurance"] = true
	wIns := envIns.do("POST", "/orders", body)
	require.Equal(t, http.StatusCreated, wIns.Code)
	var respIns map[string]any
	require.NoError(t, json.Unmarshal(wIns.Body.Bytes(), &respIns))

	assert.Greater(t, respIns["price"].(float64), respNoIns["price"].(float64),
		"price with insurance must be higher than without")
}

// ─── Get Order functional tests ───────────────────────────────────────────────

func TestFunctional_GetOrder_HappyPath(t *testing.T) {
	env := newTestEnv()

	// Create an order first
	wCreate := env.do("POST", "/orders", validCreateBody())
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var createResp map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	orderID := createResp["order_id"].(string)

	// Now retrieve it
	wGet := env.do("GET", "/orders/"+orderID, nil)
	require.Equal(t, http.StatusOK, wGet.Code)

	var getResp map[string]any
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &getResp))
	assert.Equal(t, orderID, getResp["order_id"])
	assert.Equal(t, "user-001", getResp["sender_user_id"])
	assert.Equal(t, "Bob", getResp["receiver_name"])
	assert.Equal(t, "AWAITING_PICKUP", getResp["status"])
}

func TestFunctional_GetOrder_NotFound(t *testing.T) {
	env := newTestEnv()
	w := env.do("GET", "/orders/nonexistent-id", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── List Orders functional tests ─────────────────────────────────────────────

func TestFunctional_ListOrders_HappyPath(t *testing.T) {
	env := newTestEnv()

	// Create two orders for user-001
	env.do("POST", "/orders", validCreateBody())
	env.do("POST", "/orders", validCreateBody())

	// Create one order for a different user
	otherBody := validCreateBody()
	otherBody["sender_user_id"] = "user-999"
	env.do("POST", "/orders", otherBody)

	w := env.do("GET", "/orders?user_id=user-001&page=1&limit=10", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	assert.Len(t, items, 2, "should return only user-001's orders")
}

func TestFunctional_ListOrders_EmptyResult(t *testing.T) {
	env := newTestEnv()
	w := env.do("GET", "/orders?user_id=nobody&page=1&limit=10", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	assert.Empty(t, items)
}

func TestFunctional_ListOrders_InvalidPage(t *testing.T) {
	env := newTestEnv()
	w := env.do("GET", "/orders?page=0&limit=10", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFunctional_ListOrders_InvalidStatus(t *testing.T) {
	env := newTestEnv()
	w := env.do("GET", "/orders?status=BOGUS&page=1&limit=10", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Cancel Order functional tests ────────────────────────────────────────────

func TestFunctional_CancelOrder_HappyPath(t *testing.T) {
	env := newTestEnv()

	// Create an order
	wCreate := env.do("POST", "/orders", validCreateBody())
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var createResp map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	orderID := createResp["order_id"].(string)

	// Cancel it
	wCancel := env.do("POST", "/orders/"+orderID+"/cancel", map[string]string{"reason": "changed my mind"})
	require.Equal(t, http.StatusOK, wCancel.Code)

	// Verify status changed in repo
	stored, err := env.repo.FindOrderByID(context.Background(), orderID)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusCancelled, stored.Status)
}

func TestFunctional_CancelOrder_AlreadyCancelled_Conflict(t *testing.T) {
	env := newTestEnv()

	// Create and cancel an order
	wCreate := env.do("POST", "/orders", validCreateBody())
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var createResp map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	orderID := createResp["order_id"].(string)

	env.do("POST", "/orders/"+orderID+"/cancel", map[string]string{"reason": "first cancel"})

	// Try to cancel again — should conflict
	wConflict := env.do("POST", "/orders/"+orderID+"/cancel", map[string]string{"reason": "second cancel"})
	assert.Equal(t, http.StatusConflict, wConflict.Code)
}

func TestFunctional_CancelOrder_NotFound(t *testing.T) {
	env := newTestEnv()
	w := env.do("POST", "/orders/ghost-id/cancel", map[string]string{"reason": "test"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFunctional_CancelOrder_MissingReason(t *testing.T) {
	env := newTestEnv()
	w := env.do("POST", "/orders/any-id/cancel", map[string]string{"reason": ""})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── Update Order functional tests ────────────────────────────────────────────

func TestFunctional_UpdateOrder_HappyPath(t *testing.T) {
	env := newTestEnv()

	// Create an order
	wCreate := env.do("POST", "/orders", validCreateBody())
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var createResp map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	orderID := createResp["order_id"].(string)

	// Update receiver name
	newName := "Charlie"
	wUpdate := env.do("PATCH", "/orders/"+orderID, map[string]any{"receiver_name": newName})
	require.Equal(t, http.StatusOK, wUpdate.Code)

	var updateResp map[string]any
	require.NoError(t, json.Unmarshal(wUpdate.Body.Bytes(), &updateResp))
	assert.Equal(t, orderID, updateResp["order_id"])
	assert.Equal(t, "UPDATED", updateResp["status"])

	// Verify the field was actually changed in the repo
	stored, err := env.repo.FindOrderByID(context.Background(), orderID)
	require.NoError(t, err)
	assert.Equal(t, newName, stored.ReceiverName)
}

func TestFunctional_UpdateOrder_NoUpdatableFields(t *testing.T) {
	env := newTestEnv()
	w := env.do("PATCH", "/orders/any-id", map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFunctional_UpdateOrder_Conflict_NotAwaitingPickup(t *testing.T) {
	env := newTestEnv()

	// Create and cancel an order (status → CANCELLED)
	wCreate := env.do("POST", "/orders", validCreateBody())
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var createResp map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	orderID := createResp["order_id"].(string)

	env.do("POST", "/orders/"+orderID+"/cancel", map[string]string{"reason": "cancel first"})

	// Now try to update — should conflict
	wUpdate := env.do("PATCH", "/orders/"+orderID, map[string]any{"receiver_name": "Dave"})
	assert.Equal(t, http.StatusConflict, wUpdate.Code)
}

func TestFunctional_UpdateOrder_NotFound(t *testing.T) {
	env := newTestEnv()
	w := env.do("PATCH", "/orders/ghost-id", map[string]any{"receiver_name": "Eve"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── End-to-end lifecycle test ────────────────────────────────────────────────

// TestFunctional_FullOrderLifecycle exercises the complete happy-path lifecycle:
// Create → Get → List → Update → Cancel, verifying state at each step.
func TestFunctional_FullOrderLifecycle(t *testing.T) {
	env := newTestEnv()

	// 1. Create
	wCreate := env.do("POST", "/orders", validCreateBody())
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var createResp map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	orderID := createResp["order_id"].(string)
	trackingNumber := createResp["tracking_number"].(string)
	require.NotEmpty(t, orderID)
	require.NotEmpty(t, trackingNumber)

	// 2. Get — verify initial state
	wGet := env.do("GET", "/orders/"+orderID, nil)
	require.Equal(t, http.StatusOK, wGet.Code)
	var getResp map[string]any
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &getResp))
	assert.Equal(t, "AWAITING_PICKUP", getResp["status"])
	assert.Equal(t, trackingNumber, getResp["tracking_number"])

	// 3. List — order appears in user's list
	wList := env.do("GET", "/orders?user_id=user-001&page=1&limit=10", nil)
	require.Equal(t, http.StatusOK, wList.Code)
	var listResp []map[string]any
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &listResp))
	require.Len(t, listResp, 1)
	assert.Equal(t, orderID, listResp[0]["order_id"])

	// 4. Update — change receiver name
	wUpdate := env.do("PATCH", "/orders/"+orderID, map[string]any{"receiver_name": "Charlie"})
	require.Equal(t, http.StatusOK, wUpdate.Code)

	// Verify update persisted
	stored, err := env.repo.FindOrderByID(context.Background(), orderID)
	require.NoError(t, err)
	assert.Equal(t, "Charlie", stored.ReceiverName)
	assert.Equal(t, types.OrderStatusAwaitingPickup, stored.Status,
		"status must still be AWAITING_PICKUP after update")
	assert.True(t, stored.UpdatedAt.After(time.Time{}))

	// 5. Cancel
	wCancel := env.do("POST", "/orders/"+orderID+"/cancel", map[string]string{"reason": "no longer needed"})
	require.Equal(t, http.StatusOK, wCancel.Code)

	// Verify final state
	final, err := env.repo.FindOrderByID(context.Background(), orderID)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusCancelled, final.Status)
}
