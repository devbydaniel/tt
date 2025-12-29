package config

import (
	"testing"
)

func TestConfig_GetSort(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		listName string
		want     string
	}{
		{
			name:     "empty config returns empty (code default)",
			config:   Config{},
			listName: "today",
			want:     "",
		},
		{
			name:     "global default used when no list override",
			config:   Config{Sort: "title"},
			listName: "today",
			want:     "title",
		},
		{
			name:     "list override takes precedence",
			config:   Config{Sort: "title", Today: ListSettings{Sort: "planned"}},
			listName: "today",
			want:     "planned",
		},
		{
			name:     "today list",
			config:   Config{Today: ListSettings{Sort: "planned:asc"}},
			listName: "today",
			want:     "planned:asc",
		},
		{
			name:     "upcoming list",
			config:   Config{Upcoming: ListSettings{Sort: "due"}},
			listName: "upcoming",
			want:     "due",
		},
		{
			name:     "anytime list",
			config:   Config{Anytime: ListSettings{Sort: "title"}},
			listName: "anytime",
			want:     "title",
		},
		{
			name:     "someday list",
			config:   Config{Someday: ListSettings{Sort: "created"}},
			listName: "someday",
			want:     "created",
		},
		{
			name:     "list view",
			config:   Config{List: ListSettings{Sort: "project,title"}},
			listName: "list",
			want:     "project,title",
		},
		{
			name:     "all view uses list setting",
			config:   Config{List: ListSettings{Sort: "id"}},
			listName: "all",
			want:     "id",
		},
		{
			name:     "unknown list falls back to global default",
			config:   Config{Sort: "title"},
			listName: "unknown",
			want:     "title",
		},
		{
			name:     "log list",
			config:   Config{Log: ListSettings{Sort: "created"}},
			listName: "log",
			want:     "created",
		},
		{
			name:     "inbox list",
			config:   Config{Inbox: ListSettings{Sort: "title"}},
			listName: "inbox",
			want:     "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetSort(tt.listName)
			if got != tt.want {
				t.Errorf("GetSort(%q) = %q, want %q", tt.listName, got, tt.want)
			}
		})
	}
}

func TestConfig_GetGroup(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		listName string
		want     string
	}{
		{
			name:     "empty config returns none",
			config:   Config{},
			listName: "today",
			want:     "none",
		},
		{
			name:     "global default used",
			config:   Config{Group: "project"},
			listName: "today",
			want:     "project",
		},
		{
			name:     "list override takes precedence",
			config:   Config{Group: "project", Today: ListSettings{Group: "date"}},
			listName: "today",
			want:     "date",
		},
		{
			name:     "project-list ignores global default",
			config:   Config{Group: "project"},
			listName: "project-list",
			want:     "none",
		},
		{
			name:     "project-list uses its own setting",
			config:   Config{Group: "project", ProjectList: ListSettings{Group: "area"}},
			listName: "project-list",
			want:     "area",
		},
		{
			name:     "upcoming list",
			config:   Config{Upcoming: ListSettings{Group: "date"}},
			listName: "upcoming",
			want:     "date",
		},
		{
			name:     "log list",
			config:   Config{Log: ListSettings{Group: "date"}},
			listName: "log",
			want:     "date",
		},
		{
			name:     "all view uses list setting",
			config:   Config{List: ListSettings{Group: "area"}},
			listName: "all",
			want:     "area",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetGroup(tt.listName)
			if got != tt.want {
				t.Errorf("GetGroup(%q) = %q, want %q", tt.listName, got, tt.want)
			}
		})
	}
}
