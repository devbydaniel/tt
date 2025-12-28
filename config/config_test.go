package config

import (
	"testing"
)

func TestSortingConfig_GetForCommand(t *testing.T) {
	tests := []struct {
		name   string
		config SortingConfig
		cmd    string
		want   string
	}{
		{
			name:   "empty config returns empty (code default)",
			config: SortingConfig{},
			cmd:    "today",
			want:   "",
		},
		{
			name:   "global default used when no command override",
			config: SortingConfig{Default: "title"},
			cmd:    "today",
			want:   "title",
		},
		{
			name:   "command override takes precedence",
			config: SortingConfig{Default: "title", Today: "planned"},
			cmd:    "today",
			want:   "planned",
		},
		{
			name:   "today command",
			config: SortingConfig{Today: "planned:asc"},
			cmd:    "today",
			want:   "planned:asc",
		},
		{
			name:   "upcoming command",
			config: SortingConfig{Upcoming: "due"},
			cmd:    "upcoming",
			want:   "due",
		},
		{
			name:   "anytime command",
			config: SortingConfig{Anytime: "title"},
			cmd:    "anytime",
			want:   "title",
		},
		{
			name:   "someday command",
			config: SortingConfig{Someday: "created"},
			cmd:    "someday",
			want:   "created",
		},
		{
			name:   "list command",
			config: SortingConfig{List: "project,title"},
			cmd:    "list",
			want:   "project,title",
		},
		{
			name:   "all command uses list setting",
			config: SortingConfig{List: "id"},
			cmd:    "all",
			want:   "id",
		},
		{
			name:   "unknown command falls back to default",
			config: SortingConfig{Default: "title"},
			cmd:    "unknown",
			want:   "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetForCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("GetForCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestGroupingConfig_GetForCommand(t *testing.T) {
	tests := []struct {
		name   string
		config GroupingConfig
		cmd    string
		want   string
	}{
		{
			name:   "empty config returns none",
			config: GroupingConfig{},
			cmd:    "today",
			want:   "none",
		},
		{
			name:   "global default used",
			config: GroupingConfig{Default: "project"},
			cmd:    "today",
			want:   "project",
		},
		{
			name:   "command override takes precedence",
			config: GroupingConfig{Default: "project", Today: "date"},
			cmd:    "today",
			want:   "date",
		},
		{
			name:   "project-list ignores global default",
			config: GroupingConfig{Default: "project"},
			cmd:    "project-list",
			want:   "none",
		},
		{
			name:   "project-list uses its own setting",
			config: GroupingConfig{Default: "project", ProjectList: "area"},
			cmd:    "project-list",
			want:   "area",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetForCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("GetForCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
