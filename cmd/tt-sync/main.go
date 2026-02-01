package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	areausecases "github.com/devbydaniel/tt/internal/domain/area/usecases"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
	taskusecases "github.com/devbydaniel/tt/internal/domain/task/usecases"
	"github.com/devbydaniel/tt/internal/sync/server"
	"github.com/devbydaniel/tt/internal/version"

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
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Full())
		return
	}

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

	// Create repositories for applying sync events
	taskRepo := task.NewRepository(db)
	areaRepo := area.NewRepository(db)

	// Create sync use cases
	syncEventRepo := syncevent.NewRepository(db)

	// Create applier for materializing sync events into tasks/areas tables
	applier := &usecases.ApplyEntityStates{
		TaskUpserter:     taskRepo,
		TaskByUUIDLookup: taskRepo,
		AreaByUUIDLookup: areaRepo,
		AreaUpserter:     areaRepo,
	}

	receiveEvents := &usecases.ReceiveEvents{
		Repo:    syncEventRepo,
		Applier: applier,
	}
	resetSyncEvents := &usecases.ResetSyncEvents{
		Repo: syncEventRepo,
	}

	deleteAllTasks := &taskusecases.DeleteAllTasks{Repo: taskRepo}
	deleteAllAreas := &areausecases.DeleteAllAreas{Repo: areaRepo}

	// Create handlers
	syncHandler := server.NewHandler(receiveEvents, resetSyncEvents, deleteAllTasks, deleteAllAreas)
	taskHandler := server.NewTaskHandler(application)
	areaHandler := server.NewAreaHandler(application)
	projectHandler := server.NewProjectHandler(application)

	// Set up routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", syncHandler.HandleHealth)

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Sync routes
	mux.HandleFunc("/api/v1/events", server.AuthMiddleware(cfg.Sync.APIKey, syncHandler.HandlePushEvents))
	mux.HandleFunc("/api/v1/sync", server.AuthMiddleware(cfg.Sync.APIKey, syncHandler.HandleSync))
	mux.HandleFunc("/api/v1/sync/reset", server.AuthMiddleware(cfg.Sync.APIKey, syncHandler.HandleSyncReset))

	// Task routes
	// Note: /api/v1/tasks/completed must be registered before /api/v1/tasks/ to match first
	mux.HandleFunc("/api/v1/tasks/completed", server.AuthMiddleware(cfg.Sync.APIKey, taskHandler.HandleCompletedTasks))
	mux.HandleFunc("/api/v1/tasks", server.AuthMiddleware(cfg.Sync.APIKey, taskHandler.HandleTasks))
	mux.HandleFunc("/api/v1/tasks/", server.AuthMiddleware(cfg.Sync.APIKey, taskHandler.HandleTaskByUUID))

	// Tags route
	mux.HandleFunc("/api/v1/tags", server.AuthMiddleware(cfg.Sync.APIKey, taskHandler.HandleTagsList))

	// Area routes
	mux.HandleFunc("/api/v1/areas", server.AuthMiddleware(cfg.Sync.APIKey, areaHandler.HandleAreas))
	mux.HandleFunc("/api/v1/areas/", server.AuthMiddleware(cfg.Sync.APIKey, areaHandler.HandleAreaByUUID))

	// Project routes
	mux.HandleFunc("/api/v1/projects", server.AuthMiddleware(cfg.Sync.APIKey, projectHandler.HandleProjects))
	mux.HandleFunc("/api/v1/projects/", server.AuthMiddleware(cfg.Sync.APIKey, projectHandler.HandleProjectByUUID))

	// Start server
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	fmt.Printf("tt-sync server listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
