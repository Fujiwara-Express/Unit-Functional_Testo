package http

import (
	"encoding/json"
	"net/http"

	"github.com/fujiwara-express/pricing-service/internal/domain"
	"github.com/fujiwara-express/pricing-service/internal/service"
)

type PricingHandler struct {
	service service.PricingService
}

func NewPricingHandler(svc service.PricingService) *PricingHandler {
	return &PricingHandler{service: svc}
}

func (h *PricingHandler) CalculatePrice(w http.ResponseWriter, r *http.Request) {
	// 1. Hanya terima method POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Baca body JSON dari User
	var req domain.CalculatePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 3. Panggil Mesin Service untuk hitung harga
	resp, err := h.service.CalculatePrice(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Kirim hasil balik ke User dalam format JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}