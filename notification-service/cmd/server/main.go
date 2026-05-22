package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/notification-service/internal/client"
	"github.com/notification-service/internal/config"
	httphandler "github.com/notification-service/internal/handler/http"
	"github.com/notification-service/internal/handler/http/middleware"
	kafkahandler "github.com/notification-service/internal/handler/kafka"
	"github.com/notification-service/internal/repository"
	"github.com/notification-service/internal/service"
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
	firebaseClient := client.NewFirebaseClient(cfg.FirebaseBaseURL, cfg.FirebaseAPIKey)
	sendgridClient := client.NewSendGridClient(cfg.SendGridBaseURL, cfg.SendGridAPIKey)
	whatsappClient := client.NewWhatsAppClient(cfg.WhatsAppBaseURL, cfg.WhatsAppAPIKey)

	// Service and handlers
	repo := repository.NewNotificationRepository(db)
	svc := service.NewNotificationService(repo, firebaseClient, sendgridClient, whatsappClient)
	handler := httphandler.NewNotificationHandler(svc)
	_ = kafkahandler.NewTrackingStatusHandler(svc) // wired for Kafka consumer use

	// Routes
	mux := http.NewServeMux()
	chain := func(next http.HandlerFunc) http.Handler {
		return middleware.RequestID(middleware.Auth(next))
	}
	mux.Handle("POST /notifications/send", chain(handler.SendNotification))
	mux.Handle("GET /notifications/templates", chain(handler.ListTemplates))
	mux.Handle("POST /notifications/templates", chain(handler.CreateTemplate))
	mux.Handle("PUT /notifications/templates/{template_id}", chain(handler.UpdateTemplate))

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("notification-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
