package http

import (
	"encoding/json"
	"net/http"

	"github.com/fujiwara-express/warehouse-service/internal/domain"
	"github.com/fujiwara-express/warehouse-service/internal/service"
)

type WarehouseHandler struct {
	service service.WarehouseService
}

func NewWarehouseHandler(svc service.WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{service: svc}
}

// HandleReceiveItem: Menerima request HTTP untuk barang masuk
func (h *WarehouseHandler) HandleReceiveItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var item domain.Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Format JSON tidak valid", http.StatusBadRequest)
		return
	}

	if err := h.service.ReceiveItem(r.Context(), item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Barang berhasil masuk gudang!"}`))
}

// HandleDispatchItem: Menerima request HTTP untuk barang keluar
func (h *WarehouseHandler) HandleDispatchItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.StockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Format JSON tidak valid", http.StatusBadRequest)
		return
	}

	if err := h.service.DispatchItem(r.Context(), req.ItemID, req.Quantity); err != nil {
		if err == domain.ErrOutOfStock || err == domain.ErrItemNotFound {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Barang berhasil dikeluarkan!"}`))
}

// HandleCheckStock: Menerima request HTTP untuk cek stok
func (h *WarehouseHandler) HandleCheckStock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ambil ID dari URL, contoh: /check-stock?id=BRG-001
	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		http.Error(w, "Parameter id wajib diisi!", http.StatusBadRequest)
		return
	}

	item, err := h.service.CheckStock(r.Context(), itemID)
	if err != nil {
		if err == domain.ErrItemNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}