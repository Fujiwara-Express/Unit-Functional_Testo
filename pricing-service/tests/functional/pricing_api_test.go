package functional_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
	
	// Nanti Anda akan meng-import router dan koneksi DB Anda di sini
	// "github.com/fujiwara-express/pricing-service/internal/delivery"
)

func TestCalculatePrice_API_Functional(t *testing.T) {
	// 1. (Nanti) Setup Koneksi ke Database Testing Asli
	// db := setupTestDatabase()
	
	// 2. (Nanti) Setup Router/API Endpoint Anda
	// router := delivery.SetupRouter(db)

	// 3. Membuat Request persis seperti dari Postman atau Web
	// Target Endpoint: GET /pricing/calculate
	reqURL := "/pricing/calculate?origin=BDG&destination=JKT&weight=5&service_type=REG&length=120&width=20&height=20"
	req, err := http.NewRequest("GET", reqURL, nil)
	assert.NoError(t, err)

	// 4. Menyiapkan Perekam Response (Response Recorder)
	recorder := httptest.NewRecorder()

	// 5. Menjalankan Request ke Router
	// 5. Akali Golang agar variabel req dianggap terpakai
	_ = req
	// router.ServeHTTP(recorder, req)

	// =========================================================
	// PERHATIAN: 
	// Tes di bawah ini akan FAILED untuk saat ini (Sesuai ekspektasi dosen),
	// karena router.ServeHTTP di atas masih di-comment (kodenya belum dibuat).
	// =========================================================

	// 6. Validasi Hasil (Apakah statusnya 200 OK?)
	assert.Equal(t, http.StatusOK, recorder.Code, "Ekspektasi API mengembalikan status 200 OK")

	// 7. (Opsional) Validasi isi JSON-nya nanti
	// body := recorder.Body.String()
	// assert.Contains(t, body, `"oversize_surcharge": 50000`)
}