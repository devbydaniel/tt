package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ListSettings holds per-list configuration options
type ListSettings struct {
	Sort      string `toml:"sort"`
	Group     string `toml:"group"`
	HideScope bool   `toml:"hide_scope"`
}

type Config struct {
	Database string
	Sort     string // global default sort
	Group    string // global default group
	Today         ListSettings
	Upcoming      ListSettings
	Anytime       ListSettings
	Someday       ListSettings
	Log           ListSettings
	ProjectList ListSettings
	Project     ListSettings
	Area        ListSettings
	Tag         ListSettings
	List        ListSettings // for "all" view
	Inbox         ListSettings
	Theme       ThemeConfig
}

// ThemeConfig holds color and icon settings for output formatting
type ThemeConfig struct {
	Name    string     `toml:"name"`    // preset theme name: dracula, nord, gruvbox, tokyo-night, solarized-light, catppuccin-latte
	Muted   string     `toml:"muted"`   // color for dates, tags, secondary info
	Accent  string     `toml:"accent"`  // color for planned-today indicator
	Warning string     `toml:"warning"` // color for due/overdue indicator
	Success string     `toml:"success"` // color for success messages
	Error   string     `toml:"error"`   // color for error messages
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
	Done    string `toml:"done"`    // indicator for completed tasks (default: ✓)
}

// GetSort returns the sort setting for a list view.
// Priority: list-specific > global default > "" (code default)
func (c *Config) GetSort(listName string) string {
	var listSetting string
	switch listName {
	case "today":
		listSetting = c.Today.Sort
	case "upcoming":
		listSetting = c.Upcoming.Sort
	case "anytime":
		listSetting = c.Anytime.Sort
	case "someday":
		listSetting = c.Someday.Sort
	case "log":
		listSetting = c.Log.Sort
	case "project-list":
		listSetting = c.ProjectList.Sort
	case "project":
		listSetting = c.Project.Sort
	case "area":
		listSetting = c.Area.Sort
	case "tag":
		listSetting = c.Tag.Sort
	case "list", "all":
		listSetting = c.List.Sort
	case "inbox":
		listSetting = c.Inbox.Sort
	}
	if listSetting != "" {
		return listSetting
	}
	return c.Sort // global default (empty means code default)
}

// GetGroup returns the group setting for a list view.
// Priority: list-specific > global default > "none"
func (c *Config) GetGroup(listName string) string {
	var listSetting string
	switch listName {
	case "today":
		listSetting = c.Today.Group
	case "upcoming":
		listSetting = c.Upcoming.Group
	case "anytime":
		listSetting = c.Anytime.Group
	case "someday":
		listSetting = c.Someday.Group
	case "log":
		listSetting = c.Log.Group
	case "project-list":
		listSetting = c.ProjectList.Group
	case "project":
		listSetting = c.Project.Group
	case "area":
		listSetting = c.Area.Group
	case "tag":
		listSetting = c.Tag.Group
	case "list", "all":
		listSetting = c.List.Group
	case "inbox":
		listSetting = c.Inbox.Group
	}
	if listSetting != "" {
		return listSetting
	}
	// Don't apply global default to project-list (it uses different grouping options)
	if listName == "project-list" {
		return "none"
	}
	// project/area/tag filter defaults to no grouping
	if listName == "project" || listName == "area" || listName == "tag" {
		return "none"
	}
	if c.Group != "" {
		return c.Group
	}
	return "none"
}

// GetHideScope returns the hide_scope setting for a list view.
func (c *Config) GetHideScope(listName string) bool {
	switch listName {
	case "today":
		return c.Today.HideScope
	case "upcoming":
		return c.Upcoming.HideScope
	case "anytime":
		return c.Anytime.HideScope
	case "someday":
		return c.Someday.HideScope
	case "log":
		return c.Log.HideScope
	case "project-list":
		return c.ProjectList.HideScope
	case "project":
		return c.Project.HideScope
	case "area":
		return c.Area.HideScope
	case "tag":
		return c.Tag.HideScope
	case "list", "all":
		return c.List.HideScope
	case "inbox":
		return c.Inbox.HideScope
	}
	return false
}

// fileConfig represents the TOML config file structure
type fileConfig struct {
	DataDir string `toml:"data_dir"`
	Sort    string `toml:"sort"`
	Group   string `toml:"group"`
	Today         ListSettings `toml:"today"`
	Upcoming      ListSettings `toml:"upcoming"`
	Anytime       ListSettings `toml:"anytime"`
	Someday       ListSettings `toml:"someday"`
	Log           ListSettings `toml:"log"`
	ProjectList ListSettings `toml:"project_list"`
	Project     ListSettings `toml:"project"`
	Area        ListSettings `toml:"area"`
	Tag         ListSettings `toml:"tag"`
	List        ListSettings `toml:"list"`
	Inbox         ListSettings `toml:"inbox"`
	Theme       ThemeConfig  `toml:"theme"`
}

func Load() (*Config, error) {
	dataDir := resolveDataDir()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	cfg := &Config{
		Database: filepath.Join(dataDir, "tasks.db"),
	}

	if configPath := configFilePath(); configPath != "" {
		var fc fileConfig
		if _, err := toml.DecodeFile(configPath, &fc); err == nil {
			cfg.Sort = fc.Sort
			cfg.Group = fc.Group
			cfg.Today = fc.Today
			cfg.Upcoming = fc.Upcoming
			cfg.Anytime = fc.Anytime
			cfg.Someday = fc.Someday
			cfg.Log = fc.Log
			cfg.ProjectList = fc.ProjectList
			cfg.Project = fc.Project
			cfg.Area = fc.Area
			cfg.Tag = fc.Tag
			cfg.List = fc.List
			cfg.Inbox = fc.Inbox
			cfg.Theme = fc.Theme
		}
	}

	return cfg, nil
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
