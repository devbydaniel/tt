package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Database    string
	DefaultList string // default view: today, upcoming, anytime, someday, inbox, all
	Grouping    GroupingConfig
	Theme       ThemeConfig
}

// ThemeConfig holds color and icon settings for output formatting
type ThemeConfig struct {
	Name    string     `toml:"name"`    // preset theme name: dracula, nord, gruvbox, tokyo-night, solarized-light, catppuccin-latte
	Muted   string     `toml:"muted"`   // color for dates, tags, secondary info
	Accent  string     `toml:"accent"`  // color for planned-today indicator
	Warning string     `toml:"warning"` // color for due/overdue indicator
	Header  string     `toml:"header"`  // color for section headers (bold applied automatically)
	ID      string     `toml:"id"`      // color for task IDs (empty = inherit from muted)
	Scope   string     `toml:"scope"`   // color for project/area column
	Icons   IconConfig `toml:"icons"`
}

// IconConfig holds customizable icon characters
type IconConfig struct {
	Planned string `toml:"planned"` // indicator for tasks planned today or earlier (default: ★)
	Due     string `toml:"due"`     // indicator for due/overdue tasks (default: ⚑)
	Date    string `toml:"date"`    // prefix for planned dates (default: 📅)
}

// GroupingConfig holds grouping settings with global default and per-command overrides
type GroupingConfig struct {
	Default     string `toml:"default"`      // global default: project, area, date, none
	List        string `toml:"list"`         // override for list command
	Today       string `toml:"today"`        // override for today command
	Upcoming    string `toml:"upcoming"`     // override for upcoming command
	Anytime     string `toml:"anytime"`      // override for anytime command
	Someday     string `toml:"someday"`      // override for someday command
	Log         string `toml:"log"`          // override for log command
	ProjectList string `toml:"project_list"` // override for project list command (area, none)
}

// GetForCommand returns the grouping setting for a specific command.
// Priority: command-specific > global default > "none"
func (g GroupingConfig) GetForCommand(cmd string) string {
	var cmdSetting string
	switch cmd {
	case "list":
		cmdSetting = g.List
	case "today":
		cmdSetting = g.Today
	case "upcoming":
		cmdSetting = g.Upcoming
	case "anytime":
		cmdSetting = g.Anytime
	case "someday":
		cmdSetting = g.Someday
	case "log":
		cmdSetting = g.Log
	case "project-list":
		cmdSetting = g.ProjectList
	}
	if cmdSetting != "" {
		return cmdSetting
	}
	// Don't apply global default to project-list (it uses different grouping options)
	if cmd == "project-list" {
		return "none"
	}
	if g.Default != "" {
		return g.Default
	}
	return "none"
}

// fileConfig represents the TOML config file structure
type fileConfig struct {
	DataDir     string         `toml:"data_dir"`
	DefaultList string         `toml:"default_list"`
	Grouping    GroupingConfig `toml:"grouping"`
	Theme       ThemeConfig    `toml:"theme"`
}

func Load() (*Config, error) {
	dataDir := resolveDataDir()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	// Load config from file
	var defaultList string
	var grouping GroupingConfig
	var theme ThemeConfig
	if configPath := configFilePath(); configPath != "" {
		var fc fileConfig
		if _, err := toml.DecodeFile(configPath, &fc); err == nil {
			defaultList = fc.DefaultList
			grouping = fc.Grouping
			theme = fc.Theme
		}
	}

	return &Config{
		Database:    filepath.Join(dataDir, "tasks.db"),
		DefaultList: defaultList,
		Grouping:    grouping,
		Theme:       theme,
	}, nil
}

// resolveDataDir determines the data directory with priority:
// 1. TT_DATA_DIR environment variable
// 2. Config file (~/.config/tt/config.toml)
// 3. Default (~/.local/share/tt)
func resolveDataDir() string {
	// Priority 1: Environment variable
	if envDir := os.Getenv("TT_DATA_DIR"); envDir != "" {
		return expandTilde(envDir)
	}

	// Priority 2: Config file
	if configPath := configFilePath(); configPath != "" {
		var fc fileConfig
		if _, err := toml.DecodeFile(configPath, &fc); err == nil && fc.DataDir != "" {
			return expandTilde(fc.DataDir)
		}
	}

	// Priority 3: Default
	return defaultDataDir()
}

// configFilePath returns the config file path if it exists
func configFilePath() string {
	var configDir string
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = filepath.Join(xdgConfig, "tt")
	} else if home, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(home, ".config", "tt")
	} else {
		return ""
	}

	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	return ""
}

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func defaultDataDir() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "tt")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".tt")
	}

	return filepath.Join(home, ".local", "share", "tt")
}
