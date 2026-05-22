package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"user-service/internal/config"
	"user-service/internal/repository"
	authsvc "user-service/internal/service/auth"
	usersvc "user-service/internal/service/user"
	"user-service/internal/token"
	transport "user-service/internal/transport/http"
)

func main() {
	cfg := config.Load()

	db, err := sqlx.Connect("postgres", cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	migrationsDir := getEnvOrDefault("MIGRATIONS_DIR", "migrations")
	if err := runMigrations(db, migrationsDir); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	userRepo := repository.NewPostgresUserRepository(db)
	tokenRepo := repository.NewPostgresRefreshTokenRepository(db)
	tokenMgr := token.NewJWTManager(cfg.JWTSecret)

	authService := authsvc.New(userRepo, tokenRepo, tokenMgr)
	userService := usersvc.New(userRepo)

	router := transport.NewRouter(authService, userService, tokenMgr)

	log.Printf("Starting user-service on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// runMigrations executes all *.sql files from dir in lexicographic order.
// A schema_migrations table tracks which files have already been applied.
func runMigrations(db *sqlx.DB, dir string) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename   TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, name).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			log.Printf("migration already applied: %s", name)
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		log.Printf("applied migration: %s", name)
	}

	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
