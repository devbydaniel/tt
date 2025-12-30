package output

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/config"
)

// Theme holds pre-built Lipgloss styles for consistent output formatting
type Theme struct {
	Muted   lipgloss.Style
	Accent  lipgloss.Style
	Warning lipgloss.Style
	Success lipgloss.Style
	Error   lipgloss.Style
	Header  lipgloss.Style
	ID      lipgloss.Style
	Scope   lipgloss.Style
	Icons   Icons
}

// Icons holds customizable icon characters
type Icons struct {
	Planned string
	Due     string
	Date    string
	Done    string
}

// themeColors holds the raw color values for a theme preset
type themeColors struct {
	Muted   string
	Accent  string
	Warning string
	Success string
	Error   string
	Header  string
	ID      string
	Scope   string
}

// Preset themes
var presets = map[string]themeColors{
	// Dark themes
	"dracula": {
		Muted:   "#6272a4", // Comment
		Accent:  "#f1fa8c", // Yellow
		Warning: "#ff5555", // Red
		Success: "#50fa7b", // Green
		Error:   "#ff5555", // Red
		Header:  "#bd93f9", // Purple
		ID:      "#6272a4", // Comment
		Scope:   "#8be9fd", // Cyan
	},
	"nord": {
		Muted:   "#4c566a", // Polar Night 4
		Accent:  "#ebcb8b", // Aurora Yellow
		Warning: "#bf616a", // Aurora Red
		Success: "#a3be8c", // Aurora Green
		Error:   "#bf616a", // Aurora Red
		Header:  "#81a1c1", // Frost 3
		ID:      "#4c566a", // Polar Night 4
		Scope:   "#88c0d0", // Frost 2
	},
	"gruvbox": {
		Muted:   "#928374", // Gray
		Accent:  "#fabd2f", // Yellow
		Warning: "#fb4934", // Red
		Success: "#b8bb26", // Green
		Error:   "#fb4934", // Red
		Header:  "#83a598", // Blue
		ID:      "#928374", // Gray
		Scope:   "#8ec07c", // Aqua
	},
	"tokyo-night": {
		Muted:   "#565f89", // Comment
		Accent:  "#e0af68", // Yellow
		Warning: "#f7768e", // Red
		Success: "#9ece6a", // Green
		Error:   "#f7768e", // Red
		Header:  "#7aa2f7", // Blue
		ID:      "#565f89", // Comment
		Scope:   "#7dcfff", // Cyan
	},
	// Light themes
	"solarized-light": {
		Muted:   "#93a1a1", // Base1
		Accent:  "#b58900", // Yellow
		Warning: "#dc322f", // Red
		Success: "#859900", // Green
		Error:   "#dc322f", // Red
		Header:  "#268bd2", // Blue
		ID:      "#93a1a1", // Base1
		Scope:   "#2aa198", // Cyan
	},
	"catppuccin-latte": {
		Muted:   "#9ca0b0", // Overlay0
		Accent:  "#df8e1d", // Yellow
		Warning: "#d20f39", // Red
		Success: "#40a02b", // Green
		Error:   "#d20f39", // Red
		Header:  "#1e66f5", // Blue
		ID:      "#9ca0b0", // Overlay0
		Scope:   "#179299", // Teal
	},
}

// DefaultTheme returns the default theme matching the original hardcoded values
func DefaultTheme() *Theme {
	return &Theme{
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Accent:  lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("82")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Header:  lipgloss.NewStyle().Bold(true),
		ID:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Scope:   lipgloss.NewStyle(),
		Icons: Icons{
			Planned: "★",
			Due:     "⚑",
			Date:    "›",
			Done:    "✓",
		},
	}
}

// NewTheme creates a Theme from config, using defaults for unset values
func NewTheme(cfg *config.ThemeConfig) *Theme {
	theme := DefaultTheme()

	if cfg == nil {
		return theme
	}

	// Apply preset if specified
	if preset, ok := presets[cfg.Name]; ok {
		theme.Muted = lipgloss.NewStyle().Foreground(parseColor(preset.Muted))
		theme.Accent = lipgloss.NewStyle().Foreground(parseColor(preset.Accent))
		theme.Warning = lipgloss.NewStyle().Foreground(parseColor(preset.Warning))
		theme.Success = lipgloss.NewStyle().Foreground(parseColor(preset.Success))
		theme.Error = lipgloss.NewStyle().Foreground(parseColor(preset.Error))
		theme.Header = lipgloss.NewStyle().Bold(true).Foreground(parseColor(preset.Header))
		theme.ID = lipgloss.NewStyle().Foreground(parseColor(preset.ID))
		theme.Scope = lipgloss.NewStyle().Foreground(parseColor(preset.Scope))
	}

	// Apply custom color overrides (can override preset values)
	if cfg.Muted != "" {
		theme.Muted = lipgloss.NewStyle().Foreground(parseColor(cfg.Muted))
	}
	if cfg.Accent != "" {
		theme.Accent = lipgloss.NewStyle().Foreground(parseColor(cfg.Accent))
	}
	if cfg.Warning != "" {
		theme.Warning = lipgloss.NewStyle().Foreground(parseColor(cfg.Warning))
	}
	if cfg.Header != "" {
		theme.Header = lipgloss.NewStyle().Bold(true).Foreground(parseColor(cfg.Header))
	}
	if cfg.ID != "" {
		theme.ID = lipgloss.NewStyle().Foreground(parseColor(cfg.ID))
	}
	if cfg.Scope != "" {
		theme.Scope = lipgloss.NewStyle().Foreground(parseColor(cfg.Scope))
	}
	if cfg.Success != "" {
		theme.Success = lipgloss.NewStyle().Foreground(parseColor(cfg.Success))
	}
	if cfg.Error != "" {
		theme.Error = lipgloss.NewStyle().Foreground(parseColor(cfg.Error))
	}

	// Apply icon overrides
	if cfg.Icons.Planned != "" {
		theme.Icons.Planned = cfg.Icons.Planned
	}
	if cfg.Icons.Due != "" {
		theme.Icons.Due = cfg.Icons.Due
	}
	if cfg.Icons.Date != "" {
		theme.Icons.Date = cfg.Icons.Date
	}
	if cfg.Icons.Done != "" {
		theme.Icons.Done = cfg.Icons.Done
	}

	return theme
}

// parseColor converts a color string to a Lipgloss color
// Supports ANSI codes (0-255) and hex colors (#RRGGBB)
func parseColor(s string) lipgloss.TerminalColor {
	if s == "" {
		return lipgloss.NoColor{}
	}
	return lipgloss.Color(s)
}

// AvailableThemes returns a list of available preset theme names
func AvailableThemes() []string {
	return []string{
		"dracula",
		"nord",
		"gruvbox",
		"tokyo-night",
		"solarized-light",
		"catppuccin-latte",
	}
}
