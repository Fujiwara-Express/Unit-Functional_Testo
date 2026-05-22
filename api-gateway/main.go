package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Config holds the upstream URLs for each backend service.
type Config struct {
	PricingURL   string
	WarehouseURL string
}

// NewGateway builds and returns an http.Handler that routes requests to the
// correct upstream service based on the URL path prefix:
//
//   - /pricing/*   → pricingURL  (prefix stripped before forwarding)
//   - /warehouse/* → warehouseURL (prefix stripped before forwarding)
//   - anything else → 404
//
// Accepting *url.URL values (instead of raw strings) makes the function easy
// to test: callers can point the upstreams at httptest.Server instances.
func NewGateway(pricingURL, warehouseURL *url.URL) http.Handler {
	pricingProxy := httputil.NewSingleHostReverseProxy(pricingURL)
	warehouseProxy := httputil.NewSingleHostReverseProxy(warehouseURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/pricing"):
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/pricing")
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			pricingProxy.ServeHTTP(w, r)

		case strings.HasPrefix(r.URL.Path, "/warehouse"):
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/warehouse")
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			warehouseProxy.ServeHTTP(w, r)

		default:
			http.Error(w, "Alamat tidak ditemukan di API Gateway Fujiwara Express", http.StatusNotFound)
		}
	})

	return mux
}

func main() {
	pricingURL, _ := url.Parse("http://localhost:8080")
	warehouseURL, _ := url.Parse("http://localhost:8081")

	gw := NewGateway(pricingURL, warehouseURL)

	port := ":8000"
	fmt.Printf("🚪 API Gateway menyala di http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, gw))
}
