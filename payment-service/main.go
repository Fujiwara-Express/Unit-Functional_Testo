package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"payment-service/domain"
	"payment-service/repository"
	"payment-service/service"

	_ "github.com/lib/pq"
)

// --- Stub Implementations for missing external integrations ---

// stubKafka implements domain.Kafka_Publisher for demonstration.
type stubKafka struct{}

func (k *stubKafka) Publish(ctx context.Context, topic string, event *domain.PaymentEvent) error {
	log.Printf("[STUB-KAFKA] Logging to topic %s: %+v\n", topic, event)
	return nil
}

// stubGateway implements domain.Payment_Gateway_Client for demonstration.
type stubGateway struct{}

func (g *stubGateway) Charge(ctx context.Context, req *domain.ChargeRequest) (*domain.ChargeResponse, error) {
	log.Printf("[GATEWAY-STUB] Charging %v for order %s via %s\n", req.Amount, req.OrderID, req.Method)
	return &domain.ChargeResponse{
		ExternalRef: "ext-ref-" + req.OrderID,
		Status:      "PENDING",
		VANumber:    "8881234567890",
		ExpiredAt:   time.Now().Add(24 * time.Hour),
	}, nil
}

// --- Main Application ---

func main() {
	// 1. Flags
	useKafkaStub := flag.Bool("kafka-stub", false, "Use a log-only stub instead of a real Kafka broker")
	flag.Parse()

	// 2. Configuration (using environment variables)
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "payment_db")
	
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	kafkaCAPath := getEnv("KAFKA_CA_PATH", "")
	kafkaCertPath := getEnv("KAFKA_CERT_PATH", "")
	kafkaKeyPath := getEnv("KAFKA_KEY_PATH", "")

	gatewaySecret := getEnv("GATEWAY_SECRET", "super-secret-key")
	serverPort := getEnv("PORT", "8080")

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
	
	var kafkaPublisher domain.Kafka_Publisher
	if *useKafkaStub {
		log.Println("Using Kafka STUB (log-only mode)")
		kafkaPublisher = &stubKafka{}
	} else {
		log.Printf("Connecting to Kafka brokers: %s\n", kafkaBrokers)
		
		// Load TLS config if paths are provided (for Aiven/Production)
		tlsConfig, err := repository.LoadTLSConfig(kafkaCAPath, kafkaCertPath, kafkaKeyPath)
		if err != nil {
			log.Fatalf("Failed to load Kafka TLS config: %v", err)
		}
		
		if tlsConfig != nil {
			log.Println("Kafka SSL/TLS enabled")
		}

		realKafka := repository.NewKafkaPublisher(kafkaBrokers, tlsConfig)
		defer realKafka.Close()
		kafkaPublisher = realKafka
	}

	gateway := &stubGateway{}
	validator := service.NewSignatureValidator(gatewaySecret)

	// 5. Service Initialization
	chargeSvc := service.NewChargeService(repo, gateway)
	callbackSvc := service.NewCallbackService(repo, kafkaPublisher, validator)
	codConfirmSvc := service.NewCodConfirmService(repo)
	refundSvc := service.NewRefundService(repo)

	// 6. Handler Initialization
	chargeHandler := service.NewChargeHandler(chargeSvc)
	callbackHandler := service.NewCallbackHandler(callbackSvc)
	codConfirmHandler := service.NewCodConfirmHandler(codConfirmSvc)
	refundHandler := service.NewRefundHandler(refundSvc)
	getByIDHandler := service.NewGetPaymentByIDHandler(repo)
	getByOrderHandler := service.NewGetPaymentByOrderIDHandler(repo)

	// 7. Routing
	mux := http.NewServeMux()

	// POST /payments/charge
	mux.Handle("/payments/charge", chargeHandler)
	mux.Handle("/payments/callback", callbackHandler)
	mux.Handle("/payments/cod/confirm", codConfirmHandler)
	mux.Handle("/payments/refund", refundHandler)
	
	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("order_id") != "" {
				getByOrderHandler.ServeHTTP(w, r)
			} else {
				http.Error(w, "missing order_id", http.StatusBadRequest)
			}
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	mux.HandleFunc("/payments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getByIDHandler.ServeHTTP(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 8. Start Server
	log.Printf("Payment Service starting on port %s...\n", serverPort)
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
