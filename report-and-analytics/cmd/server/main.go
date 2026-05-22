package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/report-and-analytics/internal/config"
	httphandler "github.com/report-and-analytics/internal/handler/http"
	"github.com/report-and-analytics/internal/handler/http/middleware"
	"github.com/report-and-analytics/internal/repository"
	"github.com/report-and-analytics/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Data warehouse connection (read replica / OLAP)
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Wire dependencies
	repo := repository.NewReportRepository(db)
	svc := service.NewReportService(repo)
	handler := httphandler.NewReportHandler(svc)

	// Routes — all protected by RequestID + Auth middleware
	mux := http.NewServeMux()
	chain := func(next http.HandlerFunc) http.Handler {
		return middleware.RequestID(middleware.Auth(next))
	}
	mux.Handle("GET /reports/orders", chain(handler.GetOrderReport))
	mux.Handle("GET /reports/delivery-performance", chain(handler.GetDeliveryPerformanceReport))
	mux.Handle("GET /reports/revenue", chain(handler.GetRevenueReport))
	mux.Handle("GET /reports/hub-performance", chain(handler.GetHubPerformanceReport))

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("report-and-analytics listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
