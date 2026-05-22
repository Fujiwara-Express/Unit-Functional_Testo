package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"order-service/types"
)

// Handler holds the HTTP handlers for the order service.
type Handler struct {
	svc types.OrderService
}

// New creates a new Handler with the given OrderService.
func New(svc types.OrderService) *Handler {
	return &Handler{svc: svc}
}

// writeJSON writes a JSON response with the given status code and body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// createOrderRequest is the JSON body for POST /orders.
type createOrderRequest struct {
	SenderUserID     string  `json:"sender_user_id"`
	SenderName       string  `json:"sender_name"`
	SenderAddress    string  `json:"sender_address"`
	SenderPhone      string  `json:"sender_phone"`
	SenderCityCode   string  `json:"sender_city_code"`
	ReceiverName     string  `json:"receiver_name"`
	ReceiverAddress  string  `json:"receiver_address"`
	ReceiverPhone    string  `json:"receiver_phone"`
	ReceiverCityCode string  `json:"receiver_city_code"`
	Weight           float64 `json:"weight"`
	Length           float64 `json:"length"`
	Width            float64 `json:"width"`
	Height           float64 `json:"height"`
	ServiceType      string  `json:"service_type"`
	IsCOD            bool    `json:"is_cod"`
	CODAmount        float64 `json:"cod_amount"`
	Insurance        bool    `json:"insurance"`
	ItemDescription  string  `json:"item_description"`
}

// createOrderResponse is the JSON body returned on successful order creation.
type createOrderResponse struct {
	OrderID        string    `json:"order_id"`
	TrackingNumber string    `json:"tracking_number"`
	Price          float64   `json:"price"`
	EstimatedDays  int       `json:"estimated_days"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// validServiceTypes is the set of accepted service_type values.
var validServiceTypes = map[types.ServiceType]bool{
	types.ServiceTypeREG:     true,
	types.ServiceTypeYES:     true,
	types.ServiceTypeOKE:     true,
	types.ServiceTypeSameDay: true,
}

// CreateOrder handles POST /orders.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON: "+err.Error())
		return
	}

	// Validate required fields.
	if req.SenderUserID == "" {
		writeError(w, http.StatusUnprocessableEntity, "missing required field: sender_user_id")
		return
	}
	if req.SenderName == "" {
		writeError(w, http.StatusUnprocessableEntity, "missing required field: sender_name")
		return
	}
	if req.ReceiverName == "" {
		writeError(w, http.StatusUnprocessableEntity, "missing required field: receiver_name")
		return
	}
	if req.ReceiverAddress == "" {
		writeError(w, http.StatusUnprocessableEntity, "missing required field: receiver_address")
		return
	}
	if req.ReceiverPhone == "" {
		writeError(w, http.StatusUnprocessableEntity, "missing required field: receiver_phone")
		return
	}

	// Validate service_type.
	svcType := types.ServiceType(req.ServiceType)
	if !validServiceTypes[svcType] {
		writeError(w, http.StatusUnprocessableEntity, "invalid service_type: "+req.ServiceType)
		return
	}

	// Validate COD.
	if req.IsCOD && req.CODAmount <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "cod_amount must be greater than 0 when is_cod is true")
		return
	}

	svcReq := types.CreateOrderRequest{
		SenderUserID:     req.SenderUserID,
		SenderName:       req.SenderName,
		SenderAddress:    req.SenderAddress,
		SenderPhone:      req.SenderPhone,
		SenderCityCode:   req.SenderCityCode,
		ReceiverName:     req.ReceiverName,
		ReceiverAddress:  req.ReceiverAddress,
		ReceiverPhone:    req.ReceiverPhone,
		ReceiverCityCode: req.ReceiverCityCode,
		Weight:           req.Weight,
		Length:           req.Length,
		Width:            req.Width,
		Height:           req.Height,
		ServiceType:      svcType,
		IsCOD:            req.IsCOD,
		CODAmount:        req.CODAmount,
		Insurance:        req.Insurance,
		ItemDescription:  req.ItemDescription,
	}

	resp, err := h.svc.CreateOrder(r.Context(), svcReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, createOrderResponse{
		OrderID:        resp.OrderID,
		TrackingNumber: resp.TrackingNumber,
		Price:          resp.Price,
		EstimatedDays:  resp.EstimatedDays,
		Status:         string(resp.Status),
		CreatedAt:      time.Now(),
	})
}

// ErrNotFound is a sentinel error for not-found conditions.
// Kept for backward compatibility with existing handler tests that reference handler.ErrNotFound.
var ErrNotFound = types.ErrNotFound

// ErrConflict is a sentinel error for status-conflict conditions.
// Kept for backward compatibility with existing handler tests that reference handler.ErrConflict.
var ErrConflict = types.ErrConflict

// validOrderStatuses is the set of accepted status values for list filtering.
var validOrderStatuses = map[types.OrderStatus]bool{
	types.OrderStatusCreated:        true,
	types.OrderStatusAwaitingPickup: true,
	types.OrderStatusPickedUp:       true,
	types.OrderStatusInTransit:      true,
	types.OrderStatusDelivered:      true,
	types.OrderStatusFailed:         true,
	types.OrderStatusCancelled:      true,
}

// listOrdersItem is a single order entry in the list response.
type listOrdersItem struct {
	OrderID        string  `json:"order_id"`
	TrackingNumber string  `json:"tracking_number"`
	SenderUserID   string  `json:"sender_user_id"`
	ReceiverName   string  `json:"receiver_name"`
	ServiceType    string  `json:"service_type"`
	Price          float64 `json:"price"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
}

// ListOrders handles GET /orders.
// Query params: user_id (string), status (OrderStatus), page (int >0), limit (int >0).
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	userID := q.Get("user_id")
	statusStr := q.Get("status")
	pageStr := q.Get("page")
	limitStr := q.Get("limit")

	// Validate page.
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		writeError(w, http.StatusBadRequest, "page must be a positive integer")
		return
	}

	// Validate limit.
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		writeError(w, http.StatusBadRequest, "limit must be a positive integer")
		return
	}

	// Validate status enum if provided.
	var status types.OrderStatus
	if statusStr != "" {
		status = types.OrderStatus(statusStr)
		if !validOrderStatuses[status] {
			writeError(w, http.StatusBadRequest, "invalid status: "+statusStr)
			return
		}
	}

	params := types.ListOrdersParams{
		UserID: userID,
		Status: status,
		Page:   page,
		Limit:  limit,
	}

	orders, err := h.svc.ListOrders(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error: "+err.Error())
		return
	}

	items := make([]listOrdersItem, 0, len(orders))
	for _, o := range orders {
		items = append(items, listOrdersItem{
			OrderID:        o.OrderID,
			TrackingNumber: o.TrackingNumber,
			SenderUserID:   o.SenderUserID,
			ReceiverName:   o.ReceiverName,
			ServiceType:    string(o.ServiceType),
			Price:          o.Price,
			Status:         string(o.Status),
			CreatedAt:      o.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, items)
}

// getOrderResponse is the JSON body returned for GET /orders/{order_id}.
type getOrderResponse struct {
	OrderID          string  `json:"order_id"`
	TrackingNumber   string  `json:"tracking_number"`
	SenderUserID     string  `json:"sender_user_id"`
	SenderName       string  `json:"sender_name"`
	SenderAddress    string  `json:"sender_address"`
	SenderPhone      string  `json:"sender_phone"`
	SenderCityCode   string  `json:"sender_city_code"`
	ReceiverName     string  `json:"receiver_name"`
	ReceiverAddress  string  `json:"receiver_address"`
	ReceiverPhone    string  `json:"receiver_phone"`
	ReceiverCityCode string  `json:"receiver_city_code"`
	Weight           float64 `json:"weight"`
	Length           float64 `json:"length"`
	Width            float64 `json:"width"`
	Height           float64 `json:"height"`
	ServiceType      string  `json:"service_type"`
	Price            float64 `json:"price"`
	IsCOD            bool    `json:"is_cod"`
	CODAmount        float64 `json:"cod_amount"`
	Insurance        bool    `json:"insurance"`
	ItemDescription  string  `json:"item_description"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// cancelOrderRequest is the JSON body for POST /orders/{order_id}/cancel.
type cancelOrderRequest struct {
	Reason string `json:"reason"`
}

// CancelOrder handles POST /orders/{order_id}/cancel.
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")

	var req cancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON: "+err.Error())
		return
	}

	if req.Reason == "" {
		writeError(w, http.StatusUnprocessableEntity, "missing required field: reason")
		return
	}

	err := h.svc.CancelOrder(r.Context(), orderID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "order cannot be cancelled: "+err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found: "+orderID)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// updateOrderRequest is the JSON body for PATCH /orders/{order_id}.
// All fields are optional pointers; at least one must be non-nil.
type updateOrderRequest struct {
	ReceiverName     *string `json:"receiver_name"`
	ReceiverAddress  *string `json:"receiver_address"`
	ReceiverPhone    *string `json:"receiver_phone"`
	ReceiverCityCode *string `json:"receiver_city_code"`
	ItemDescription  *string `json:"item_description"`
}

// UpdateOrder handles PATCH /orders/{order_id}.
func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")

	var req updateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON: "+err.Error())
		return
	}

	// Require at least one updatable field.
	if req.ReceiverName == nil && req.ReceiverAddress == nil && req.ReceiverPhone == nil &&
		req.ReceiverCityCode == nil && req.ItemDescription == nil {
		writeError(w, http.StatusUnprocessableEntity, "no updatable fields provided")
		return
	}

	svcReq := types.UpdateOrderRequest{
		ReceiverName:     req.ReceiverName,
		ReceiverAddress:  req.ReceiverAddress,
		ReceiverPhone:    req.ReceiverPhone,
		ReceiverCityCode: req.ReceiverCityCode,
		ItemDescription:  req.ItemDescription,
	}

	err := h.svc.UpdateOrder(r.Context(), orderID, svcReq)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "order cannot be updated: "+err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found: "+orderID)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"order_id": orderID, "status": "UPDATED"})
}

// GetOrder handles GET /orders/{order_id}.
// It expects the order_id to be provided via the "order_id" path value (Go 1.22+ r.PathValue).
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "missing required path parameter: order_id")
		return
	}

	order, err := h.svc.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found: "+orderID)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, getOrderResponse{
		OrderID:          order.OrderID,
		TrackingNumber:   order.TrackingNumber,
		SenderUserID:     order.SenderUserID,
		SenderName:       order.SenderName,
		SenderAddress:    order.SenderAddress,
		SenderPhone:      order.SenderPhone,
		SenderCityCode:   order.SenderCityCode,
		ReceiverName:     order.ReceiverName,
		ReceiverAddress:  order.ReceiverAddress,
		ReceiverPhone:    order.ReceiverPhone,
		ReceiverCityCode: order.ReceiverCityCode,
		Weight:           order.Weight,
		Length:           order.Length,
		Width:            order.Width,
		Height:           order.Height,
		ServiceType:      string(order.ServiceType),
		Price:            order.Price,
		IsCOD:            order.IsCOD,
		CODAmount:        order.CODAmount,
		Insurance:        order.Insurance,
		ItemDescription:  order.ItemDescription,
		Status:           string(order.Status),
		CreatedAt:        order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        order.UpdatedAt.Format(time.RFC3339),
	})
}
