package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/pickup-service/internal/client"
	"github.com/pickup-service/internal/config"
	httphandler "github.com/pickup-service/internal/handler/http"
	"github.com/pickup-service/internal/handler/http/middleware"
	"github.com/pickup-service/internal/repository"
	"github.com/pickup-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Clients
	deliveryClient := client.New(cfg.DeliveryServiceURL)
	notificationClient := client.NewHTTPNotificationClient(cfg.NotificationServiceURL)
	trackingClient := client.NewKafkaTrackingClient(cfg.TrackingKafkaTopic)

	// Service and handler
	repo := repository.NewPickupRepository(db)
	svc := service.NewPickupService(repo, deliveryClient, trackingClient, notificationClient)
	handler := httphandler.NewPickupHandler(svc)

	// Routes
	mux := http.NewServeMux()
	chain := func(next http.HandlerFunc) http.Handler {
		return middleware.RequestID(middleware.Auth(next))
	}
	mux.Handle("POST /pickups", chain(handler.RequestPickup))
	mux.Handle("POST /pickups/{pickup_id}/assign", chain(handler.AssignCourier))
	mux.Handle("POST /pickups/{pickup_id}/status", chain(handler.UpdatePickupStatus))
	mux.Handle("GET /pickups/{pickup_id}", chain(handler.GetPickup))
	mux.Handle("GET /pickups", chain(handler.ListPickups))
	mux.Handle("POST /pickups/{pickup_id}/cancel", chain(handler.CancelPickup))

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("pickup-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
