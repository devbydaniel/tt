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
	ID          int64
	UUID        string
	Title       string
	Description *string
	ProjectID   *int64
	AreaID      *int64
	PlannedDate *time.Time
	DueDate     *time.Time
	State       string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time

	// Recurrence fields
	RecurType     *string    // "fixed" or "relative"
	RecurRule     *string    // JSON rule: {"interval":1,"unit":"week",...}
	RecurEnd      *time.Time // optional end date
	RecurPaused   bool       // true = paused
	RecurParentID *int64     // links to original recurring task

	// Tags
	Tags []string

	// Display fields (populated by queries with JOINs, not persisted)
	ProjectName *string
	AreaName    *string
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
