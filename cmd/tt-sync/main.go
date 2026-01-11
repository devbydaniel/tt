package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/sync/server"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/devbydaniel/tt/docs" // swagger docs
)

// @title TT Sync Server API
// @version 1.0
// @description Task management sync server with CRUD operations
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}" where {token} is your API key

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

	// Get server client ID from environment
	serverClientID := os.Getenv("TT_SYNC_CLIENT_ID")
	if serverClientID == "" {
		serverClientID = "server"
	}

	// Create app with server clientID (no syncCfg - server doesn't sync to itself)
	application := app.New(db, serverClientID, nil)

	// Create sync use cases
	syncEventRepo := syncevent.NewRepository(db)
	receiveEvents := &usecases.ReceiveEvents{
		Repo: syncEventRepo,
	}

	// Create handlers
	syncHandler := server.NewHandler(receiveEvents)
	taskHandler := server.NewTaskHandler(application)

	// Set up routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", syncHandler.HandleHealth)

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Sync routes
	mux.HandleFunc("/api/v1/events", server.AuthMiddleware(cfg.Sync.APIKey, syncHandler.HandlePushEvents))
	mux.HandleFunc("/api/v1/sync", server.AuthMiddleware(cfg.Sync.APIKey, syncHandler.HandleSync))

	// Task routes
	mux.HandleFunc("/api/v1/tasks", server.AuthMiddleware(cfg.Sync.APIKey, taskHandler.HandleTasks))
	mux.HandleFunc("/api/v1/tasks/", server.AuthMiddleware(cfg.Sync.APIKey, taskHandler.HandleTaskByUUID))

	// Start server
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	fmt.Printf("tt-sync server listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
