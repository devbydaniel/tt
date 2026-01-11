package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/devbydaniel/tt/config"
	"github.com/joho/godotenv"
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/sync/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load .env file from working directory (optional, errors ignored)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Validate required config
	if cfg.Sync.APIKey == "" {
		return fmt.Errorf("sync.api_key must be set (via TT_SYNC_API_KEY env var, .env file, or config.toml)")
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// Create use cases
	syncEventRepo := syncevent.NewRepository(db)
	receiveEvents := &usecases.ReceiveEvents{
		Repo: syncEventRepo,
	}

	// Create handler
	handler := server.NewHandler(receiveEvents)

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HandleHealth)
	mux.HandleFunc("/api/v1/events", server.AuthMiddleware(cfg.Sync.APIKey, handler.HandlePushEvents))
	mux.HandleFunc("/api/v1/sync", server.AuthMiddleware(cfg.Sync.APIKey, handler.HandleSync))

	// Start server
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	fmt.Printf("tt-sync server listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
