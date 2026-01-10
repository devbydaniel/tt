package main

import (
	"fmt"
	"os"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/cli"
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/output"
)

func main() {
	if err := run(); err != nil {
		formatter := output.NewFormatter(os.Stderr, nil)
		formatter.Error(err.Error())
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

	var syncCfg *app.SyncConfig
	if cfg.Sync.URL != "" {
		syncCfg = &app.SyncConfig{
			URL:    cfg.Sync.URL,
			APIKey: cfg.Sync.APIKey,
		}
	}
	application := app.New(db, cfg.ClientID, syncCfg)
	theme := output.NewTheme(&cfg.Theme)

	deps := &cli.Dependencies{
		App:    application,
		Config: cfg,
		Theme:  theme,
	}

	return cli.NewRootCmd(deps).Execute()
}
