package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"delivery-service/domain"
	"delivery-service/repository"
	"delivery-service/service"

	_ "github.com/lib/pq"
)

// --- Stub Implementation for missing external integrations ---

// stubRouting implements domain.Routing_Client for demonstration / local dev.
type stubRouting struct{}

func (r *stubRouting) GetCourierRoute(_ context.Context, courierID string) (*domain.DeliveryRoute, error) {
	log.Printf("[STUB-ROUTING] GetCourierRoute called for courier %s\n", courierID)
	return &domain.DeliveryRoute{
		TotalStops:                    0,
		TotalDistanceKm:               0,
		EstimatedTotalDurationMinutes: 0,
		OptimizedRoute:                []domain.RouteStop{},
	}, nil
}

// --- Main Application ---

func main() {
	// 1. Flags
	useRoutingStub := flag.Bool("routing-stub", false, "Use a log-only stub instead of a real routing service")
	flag.Parse()

	// 2. Configuration (environment variables with sensible defaults)
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "delivery_db")

	serverPort := getEnv("PORT", "8081")

	// 3. Database Connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// 4. Dependency Injection
	repo := repository.NewPostgresRepository(db)

	var routingClient domain.Routing_Client
	if *useRoutingStub {
		log.Println("Using Routing STUB (log-only mode)")
		routingClient = &stubRouting{}
	} else {
		// Real routing client would be wired here (e.g. gRPC or HTTP client).
		// For now, fall back to the stub so the service starts without external deps.
		log.Println("No real routing client configured — falling back to stub")
		routingClient = &stubRouting{}
	}

	// 5. Service Initialization
	courierSvc := service.NewCourierService(repo)
	assignSvc := service.NewAssignService(repo, routingClient)
	statusSvc := service.NewStatusService(repo)
	querySvc := service.NewQueryService(repo)

	// 6. Handler Initialization
	registerCourierHandler := service.NewRegisterCourierHandler(courierSvc)
	updateCourierHandler := service.NewUpdateCourierHandler(courierSvc)
	listCouriersHandler := service.NewListCouriersHandler(courierSvc)
	assignHandler := service.NewAssignHandler(assignSvc)
	statusUpdateHandler := service.NewStatusUpdateHandler(statusSvc)
	getCourierJobsHandler := service.NewGetCourierJobsHandler(querySvc)
	getDeliveryDetailHandler := service.NewGetDeliveryDetailHandler(querySvc)

	// 7. Routing
	mux := http.NewServeMux()

	// POST /delivery/assign
	mux.Handle("/delivery/assign", assignHandler)

	// POST /delivery/status
	mux.Handle("/delivery/status", statusUpdateHandler)

	// POST   /delivery/couriers        → register
	// GET    /delivery/couriers        → list
	// PATCH  /delivery/couriers/{id}   → update
	mux.HandleFunc("/delivery/couriers/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			updateCourierHandler.ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/delivery/couriers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			registerCourierHandler.ServeHTTP(w, r)
		case http.MethodGet:
			listCouriersHandler.ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET /delivery/courier/{courier_id}/jobs
	mux.HandleFunc("/delivery/courier/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getCourierJobsHandler.ServeHTTP(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET /delivery/{delivery_id}
	mux.HandleFunc("/delivery/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getDeliveryDetailHandler.ServeHTTP(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 8. Start Server
	log.Printf("Delivery Service starting on port %s...\n", serverPort)
	if err := http.ListenAndServe(":"+serverPort, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
