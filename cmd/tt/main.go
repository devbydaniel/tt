package main

import (
	"fmt"
	"os"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/cli"
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	areaRepo := area.NewRepository(db)
	areaService := area.NewService(areaRepo)

	projectRepo := project.NewRepository(db)
	projectService := project.NewService(projectRepo, areaService)

	taskRepo := task.NewRepository(db)
	taskService := task.NewService(taskRepo, projectService, areaService)

	theme := output.NewTheme(&cfg.Theme)

	deps := &cli.Dependencies{
		TaskService:    taskService,
		AreaService:    areaService,
		ProjectService: projectService,
		Config:         cfg,
		Theme:          theme,
	}

	return cli.NewRootCmd(deps).Execute()
}
