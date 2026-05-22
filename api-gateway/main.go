package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func main() {
	// 1. Catat alamat ruangan masing-masing service
	pricingURL, _ := url.Parse("http://localhost:8080")
	warehouseURL, _ := url.Parse("http://localhost:8081")

	// 2. Buat mesin "Kurir" (Proxy) untuk meneruskan request
	pricingProxy := httputil.NewSingleHostReverseProxy(pricingURL)
	warehouseProxy := httputil.NewSingleHostReverseProxy(warehouseURL)

	// 3. Resepsionis (Handler) mulai bekerja
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Jika orang datang menyebut kata "/pricing" -> Arahkan ke Service Pricing
		if strings.HasPrefix(r.URL.Path, "/pricing") {
			r.URL.Path = strings.Replace(r.URL.Path, "/pricing", "", 1)
			pricingProxy.ServeHTTP(w, r)
			return
		}

		// Jika orang datang menyebut kata "/warehouse" -> Arahkan ke Service Gudang
		if strings.HasPrefix(r.URL.Path, "/warehouse") {
			r.URL.Path = strings.Replace(r.URL.Path, "/warehouse", "", 1)
			warehouseProxy.ServeHTTP(w, r)
			return
		}

		// Jika rute tidak dikenali
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Alamat tidak ditemukan di API Gateway Fujiwara Express"))
	})

	// 4. Nyalakan Resepsionis di Pintu Utama (Port 8000)
	port := ":8000"
	fmt.Printf("🚪 API Gateway menyala di http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
