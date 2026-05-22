package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"delivery-service/domain"
)

// NotFoundError is returned when a requested resource does not exist.
type NotFoundError struct{ ID string }

func (e *NotFoundError) Error() string { return "not found: " + e.ID }

// CourierService handles courier business logic.
type CourierService struct {
	repo domain.Delivery_Repository
}

// NewCourierService creates a new CourierService.
func NewCourierService(repo domain.Delivery_Repository) *CourierService {
	return &CourierService{repo: repo}
}

// newDeliveryID generates a random hex ID.
func newDeliveryID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// --- Register Courier ---

// RegisterCourierHandler is the HTTP handler for POST /delivery/couriers.
type RegisterCourierHandler struct {
	svc *CourierService
}

// NewRegisterCourierHandler creates a new RegisterCourierHandler.
func NewRegisterCourierHandler(svc *CourierService) *RegisterCourierHandler {
	return &RegisterCourierHandler{svc: svc}
}

func (h *RegisterCourierHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterCourierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Phone == "" || req.HubID == "" || req.VehicleType == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	courier := &domain.Courier{
		CourierID:   newDeliveryID(),
		Name:        req.Name,
		Phone:       req.Phone,
		HubID:       req.HubID,
		VehicleType: req.VehicleType,
		IsAvailable: true,
	}

	if err := h.svc.repo.CreateCourier(r.Context(), courier); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"courier_id": courier.CourierID,
		"status":     "CREATED",
	})
}

// --- Update Courier ---

// UpdateCourierHandler is the HTTP handler for PATCH /delivery/couriers/{courier_id}.
type UpdateCourierHandler struct {
	svc *CourierService
}

// NewUpdateCourierHandler creates a new UpdateCourierHandler.
func NewUpdateCourierHandler(svc *CourierService) *UpdateCourierHandler {
	return &UpdateCourierHandler{svc: svc}
}

func (h *UpdateCourierHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract courier_id from path: /delivery/couriers/{courier_id}
	courierID := strings.TrimPrefix(r.URL.Path, "/delivery/couriers/")
	if courierID == "" {
		http.Error(w, "missing courier_id", http.StatusBadRequest)
		return
	}

	var update domain.CourierUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.svc.repo.UpdateCourier(r.Context(), courierID, &update); err != nil {
		switch err.(type) {
		case *NotFoundError:
			http.Error(w, "courier not found", http.StatusNotFound)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"courier_id": courierID,
		"status":     "UPDATED",
	})
}

// --- List Couriers ---

// ListCouriersHandler is the HTTP handler for GET /delivery/couriers.
type ListCouriersHandler struct {
	svc *CourierService
}

// NewListCouriersHandler creates a new ListCouriersHandler.
func NewListCouriersHandler(svc *CourierService) *ListCouriersHandler {
	return &ListCouriersHandler{svc: svc}
}

func (h *ListCouriersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := &domain.CourierFilter{
		HubID:    q.Get("hub_id"),
		CityCode: q.Get("city_code"),
	}

	if v := q.Get("is_available"); v != "" {
		b := v == "true"
		filter.IsAvailable = &b
	}

	couriers, err := h.svc.repo.ListCouriers(r.Context(), filter)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if couriers == nil {
		couriers = []*domain.Courier{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(couriers)
}

// RegisterCourierWithContext is a helper used by functional tests.
func RegisterCourierWithContext(ctx context.Context, repo domain.Delivery_Repository, req *domain.RegisterCourierRequest) (*domain.Courier, error) {
	courier := &domain.Courier{
		CourierID:   newDeliveryID(),
		Name:        req.Name,
		Phone:       req.Phone,
		HubID:       req.HubID,
		VehicleType: req.VehicleType,
		IsAvailable: true,
	}
	if err := repo.CreateCourier(ctx, courier); err != nil {
		return nil, err
	}
	return courier, nil
}
