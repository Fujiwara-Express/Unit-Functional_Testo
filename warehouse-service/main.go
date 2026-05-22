package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	handler "github.com/fujiwara-express/warehouse-service/internal/delivery/http"
	"github.com/fujiwara-express/warehouse-service/internal/repository"
	"github.com/fujiwara-express/warehouse-service/internal/service"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

func main() {
	// 1. Konfigurasi Koneksi Database PostgreSQL
	// Karena Atala belum acc, sementara pakai database lokal Anda dulu atau persiapkan string-nya
	connStr := "user=postgres password=rahasia dbname=fujiwara_warehouse sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database gudang: %v", err)
	}
	defer db.Close()

	// Tes koneksi ke database gudang
	if err := db.Ping(); err != nil {
		log.Printf("⚠️  Database Gudang lokal belum tersambung (Menunggu jaringan/DB lokal): %v\n", err)
	} else {
		fmt.Println("✅ Warehouse Service berhasil terhubung ke PostgreSQL!")
	}

	// 2. Inisialisasi Repository menggunakan POSTGRES
	//repo := repository.NewPostgresRepository(db)
	repo := repository.NewMemoryRepository()
	// 3. Inisialisasi Service (Kepala Gudang)
	svc := service.NewWarehouseService(repo)

	// 4. Inisialisasi Handler (Resepsionis API)
	hdl := handler.NewWarehouseHandler(svc)

	// 5. Daftarkan Alamat API (Routing)
	http.HandleFunc("/receive", hdl.HandleReceiveItem)
	http.HandleFunc("/dispatch", hdl.HandleDispatchItem)
	http.HandleFunc("/check-stock", hdl.HandleCheckStock)

	// 6. Nyalakan Server
	port := ":8081" // Port 8081 agar tidak bentrok dengan Pricing (8080)
	fmt.Printf("📦 Warehouse Service menyala di http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}