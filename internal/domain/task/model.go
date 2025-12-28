package task

import (
	"fmt"
	"strings"
	"time"
)

// SortField represents a field that can be sorted
type SortField string

const (
	SortByID      SortField = "id"
	SortByTitle   SortField = "title"
	SortByPlanned SortField = "planned"
	SortByDue     SortField = "due"
	SortByCreated SortField = "created"
	SortByProject SortField = "project"
	SortByArea    SortField = "area"
)

// ValidSortFields returns all valid sort field names
func ValidSortFields() []string {
	return []string{"id", "title", "planned", "due", "created", "project", "area"}
}

// SortDirection represents ascending or descending order
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortOption represents a single sort criterion
type SortOption struct {
	Field     SortField
	Direction SortDirection
}

// DefaultSort returns the default sort options (created desc)
func DefaultSort() []SortOption {
	return []SortOption{{Field: SortByCreated, Direction: SortDesc}}
}

// ParseSort parses a sort string into SortOptions
// Format: "field:dir,field:dir" e.g. "due:asc,title:desc" or just "due,title"
// Default direction: desc for date fields (planned, due, created), asc for others
func ParseSort(s string) ([]SortOption, error) {
	if s == "" {
		return DefaultSort(), nil
	}

	parts := strings.Split(s, ",")
	opts := make([]SortOption, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var field, dir string
		if idx := strings.Index(part, ":"); idx != -1 {
			field = part[:idx]
			dir = part[idx+1:]
		} else {
			field = part
			dir = ""
		}

		sortField, err := parseSortField(field)
		if err != nil {
			return nil, err
		}

		sortDir := defaultDirection(sortField)
		if dir != "" {
			sortDir, err = parseSortDirection(dir)
			if err != nil {
				return nil, err
			}
		}

		opts = append(opts, SortOption{Field: sortField, Direction: sortDir})
	}

	if len(opts) == 0 {
		return DefaultSort(), nil
	}

	return opts, nil
}

func parseSortField(s string) (SortField, error) {
	switch strings.ToLower(s) {
	case "id":
		return SortByID, nil
	case "title":
		return SortByTitle, nil
	case "planned":
		return SortByPlanned, nil
	case "due":
		return SortByDue, nil
	case "created":
		return SortByCreated, nil
	case "project":
		return SortByProject, nil
	case "area":
		return SortByArea, nil
	default:
		return "", fmt.Errorf("invalid sort field: %q (valid: %s)", s, strings.Join(ValidSortFields(), ", "))
	}
}

func parseSortDirection(s string) (SortDirection, error) {
	switch strings.ToLower(s) {
	case "asc":
		return SortAsc, nil
	case "desc":
		return SortDesc, nil
	default:
		return "", fmt.Errorf("invalid sort direction: %q (valid: asc, desc)", s)
	}
}

func defaultDirection(f SortField) SortDirection {
	switch f {
	case SortByPlanned, SortByDue, SortByCreated:
		return SortDesc
	default:
		return SortAsc
	}
}

type Task struct {
	ID          int64      `json:"id"`
	UUID        string     `json:"uuid"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	ProjectID   *int64     `json:"projectId,omitempty"`
	AreaID      *int64     `json:"areaId,omitempty"`
	PlannedDate *time.Time `json:"plannedDate,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	State       string     `json:"state"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Recurrence fields
	RecurType     *string    `json:"recurType,omitempty"`     // "fixed" or "relative"
	RecurRule     *string    `json:"recurRule,omitempty"`     // JSON rule: {"interval":1,"unit":"week",...}
	RecurEnd      *time.Time `json:"recurEnd,omitempty"`      // optional end date
	RecurPaused   bool       `json:"recurPaused,omitempty"`   // true = paused
	RecurParentID *int64     `json:"recurParentId,omitempty"` // links to original recurring task

	// Tags
	Tags []string `json:"tags,omitempty"`

	// Display fields (populated by queries with JOINs, not persisted)
	ProjectName *string `json:"projectName,omitempty"`
	AreaName    *string `json:"areaName,omitempty"`
}

// Recurrence type constants
const (
	RecurTypeFixed    = "fixed"
	RecurTypeRelative = "relative"
)

const (
	StatusTodo = "todo"
	StatusDone = "done"

	StateActive  = "active"
	StateSomeday = "someday"
)
