package note

import (
	"errors"
	"time"
)

// EntityType identifies which kind of entity a note is attached to.
type EntityType string

const (
	EntityTask    EntityType = "task"
	EntityProject EntityType = "project"
	EntityArea    EntityType = "area"
)

// ValidEntityType reports whether the given entity type is recognized.
func ValidEntityType(et EntityType) bool {
	switch et {
	case EntityTask, EntityProject, EntityArea:
		return true
	}
	return false
}

// ErrNoteNotFound is returned when a note cannot be located on disk.
var ErrNoteNotFound = errors.New("note not found")

// Note represents a single markdown note attached to a task, project, or area.
//
// Notes live on the filesystem (not in SQLite) under:
//
//	<notes_dir>/<entity_type>/<entity_uuid>/YYYYMMDD--<slug>.md
//
// The Path field is the absolute filesystem path. The other fields are derived
// from the path and filename for convenience and JSON output.
type Note struct {
	Path       string     `json:"path"`
	Filename   string     `json:"filename"`
	Title      string     `json:"title"`
	Date       time.Time  `json:"date"`
	EntityType EntityType `json:"entityType"`
	EntityUUID string     `json:"entityUuid"`

	// EntityName and EntityID are populated by the CLI layer when listing,
	// after resolving the UUID against the task/area domain. They are not
	// derived from the filesystem.
	EntityName string `json:"entityName,omitempty"`
	EntityID   int64  `json:"entityId,omitempty"`
}

// SearchMatch represents a single line match from a notes search.
type SearchMatch struct {
	Note    Note   `json:"note"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}
