package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Database string
}

func Load() (*Config, error) {
	dataDir := defaultDataDir()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &Config{
		Database: filepath.Join(dataDir, "tasks.db"),
	}, nil
}

func defaultDataDir() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "t")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".t")
	}

	return filepath.Join(home, ".local", "share", "t")
}
