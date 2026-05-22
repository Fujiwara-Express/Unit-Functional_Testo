package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	handler "github.com/fujiwara-express/pricing-service/internal/delivery/http"
	"github.com/fujiwara-express/pricing-service/internal/repository"
	"github.com/fujiwara-express/pricing-service/internal/service"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

func main() {
	// 1. Konfigurasi Koneksi Database PostgreSQL
	// Sesuaikan user, password, dan dbname dengan yang ada di komputer Anda!
	// Menggunakan IP dan database dari Atala lewat ZeroTier
connStr := "host=192.168.192.101 port=5432 user=elcloud password=cloud2026 dbname=fujiwara_pricing sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	// Tes koneksi apakah database menyala
	// if err := db.Ping(); err != nil {
	// 	log.Fatalf("Database PostgreSQL tidak merespons: %v\n(Pastikan PostgreSQL sudah menyala dan nama database/password benar)", err)
	// }
	fmt.Println("✅ Berhasil terhubung ke database PostgreSQL!")

	// 2. Inisialisasi Repository (Sekarang pakai POSTGRES, bukan Memory lagi!)
	//repo := repository.NewPostgresRepository(db)
	repo := repository.NewMemoryRepository()
	// 3. Inisialisasi Service (Logika Bisnis)
	svc := service.NewPricingService(repo)

	// 4. Inisialisasi Handler (Penerima Request HTTP)
	hdl := handler.NewPricingHandler(svc)

	// 5. Daftarkan Alamat API
	http.HandleFunc("/calculate-price", hdl.CalculatePrice)

	// 6. Nyalakan Server
	port := ":8080"
	fmt.Printf("✅ Pricing Service menyala di http://localhost%s\n", port)
	
	log.Fatal(http.ListenAndServe(port, nil))
}