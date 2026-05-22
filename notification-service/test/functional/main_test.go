package functional_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/notification-service/internal/client"
	httphandler "github.com/notification-service/internal/handler/http"
	"github.com/notification-service/internal/handler/http/middleware"
	kafkahandler "github.com/notification-service/internal/handler/kafka"
	"github.com/notification-service/internal/repository"
	"github.com/notification-service/internal/service"
)

var (
	testDB       *sql.DB
	testServer   *httptest.Server
	firebaseStub *firebaseStubServer
	sendgridStub *sendgridStubServer
	whatsappStub *whatsappStubServer
	kafkaHandler *kafkahandler.TrackingStatusHandler
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start Postgres container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "notif_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	host, err := pgContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		os.Exit(1)
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=test password=test dbname=notif_test sslmode=disable",
		host, port.Port(),
	)

	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db: %v\n", err)
		os.Exit(1)
	}
	if err := testDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping db: %v\n", err)
		os.Exit(1)
	}

	if err := applySchema(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply schema: %v\n", err)
		os.Exit(1)
	}

	// Build stubs
	firebaseStub = newFirebaseStub()
	sendgridStub = newSendGridStub()
	whatsappStub = newWhatsAppStub()
	defer firebaseStub.Close()
	defer sendgridStub.Close()
	defer whatsappStub.Close()

	// Wire SUT
	repo := repository.NewNotificationRepository(testDB)
	svc := service.NewNotificationService(
		repo,
		client.NewFirebaseClient(firebaseStub.URL(), "test-key"),
		client.NewSendGridClient(sendgridStub.URL(), "test-key"),
		client.NewWhatsAppClient(whatsappStub.URL(), "test-key"),
	)
	handler := httphandler.NewNotificationHandler(svc)
	kafkaHandler = kafkahandler.NewTrackingStatusHandler(svc)

	mux := buildMux(handler)
	testServer = httptest.NewServer(mux)
	defer testServer.Close()

	os.Exit(m.Run())
}

// applySchema creates the required tables in the test database.
func applySchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notification_templates (
			template_id   TEXT PRIMARY KEY,
			event_type    TEXT NOT NULL,
			channel       TEXT NOT NULL,
			subject       TEXT NOT NULL DEFAULT '',
			body_template TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS notification_logs (
			notif_id        TEXT PRIMARY KEY,
			user_id         TEXT NOT NULL,
			tracking_number TEXT NOT NULL DEFAULT '',
			channel         TEXT NOT NULL,
			template_id     TEXT NOT NULL,
			message         TEXT NOT NULL DEFAULT '',
			status          TEXT NOT NULL,
			sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

// buildMux registers all four notification routes wrapped with RequestID and Auth middleware.
func buildMux(h *httphandler.NotificationHandler) *http.ServeMux {
	mux := http.NewServeMux()
	chain := func(next http.HandlerFunc) http.Handler {
		return middleware.RequestID(middleware.Auth(next))
	}
	mux.Handle("POST /notifications/send", chain(h.SendNotification))
	mux.Handle("GET /notifications/templates", chain(h.ListTemplates))
	mux.Handle("POST /notifications/templates", chain(h.CreateTemplate))
	mux.Handle("PUT /notifications/templates/{template_id}", chain(h.UpdateTemplate))
	return mux
}
