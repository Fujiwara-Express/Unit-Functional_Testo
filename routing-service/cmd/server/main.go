package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"routing-service/internal/clients"
	"routing-service/internal/handlers"
	"routing-service/internal/middleware"
	"routing-service/internal/repositories"
	"routing-service/internal/services"
)

func main() {
	ctx := context.Background()

	// ── PostgreSQL ────────────────────────────────────────────────────────────
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/routing?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	// ── Dependencies ──────────────────────────────────────────────────────────
	repo := repositories.NewGraphRepository(pool)
	cache := services.NewCacheService(rdb)

	deliveryBaseURL := os.Getenv("DELIVERY_SERVICE_URL")
	if deliveryBaseURL == "" {
		deliveryBaseURL = "http://localhost:8081"
	}
	deliveryClient := clients.NewDeliveryServiceClient(deliveryBaseURL, 5*time.Second)

	nodesHandler := handlers.NewNodesHandler(repo)
	edgesHandler := handlers.NewEdgesHandler(repo, cache)
	routeHandler := handlers.NewRouteHandler(repo, cache)
	courierHandler := handlers.NewCourierHandler(deliveryClient, cache)

	// ── Router ────────────────────────────────────────────────────────────────
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())

	routing := r.Group("/routing")
	{
		routing.GET("/route", routeHandler.GetRoute)
		routing.GET("/nodes", nodesHandler.GetNodes)
		routing.POST("/nodes", nodesHandler.CreateNode)
		routing.GET("/edges", edgesHandler.GetEdges)
		routing.POST("/edges", edgesHandler.CreateEdge)
		routing.PATCH("/edges/:edge_id", edgesHandler.UpdateEdge)
		routing.GET("/courier-route/:courier_id", courierHandler.GetCourierRoute)
	}

	// ── Server ────────────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("routing-service listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}
