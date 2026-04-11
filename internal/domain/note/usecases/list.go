package usecases

import (
	"fmt"
	"time"

	"github.com/devbydaniel/tt/internal/domain/note"
)

// parseDateFlag parses a YYYY-MM-DD string into a local-midnight time.Time.
// Returns the zero value if value is empty.
func ParseDateFlag(flag, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --%s date (expected YYYY-MM-DD): %w", flag, err)
	}
	return t, nil
}

// ListNotes returns notes for a given entity, or all notes if EntityType is empty.
type ListNotes struct {
	Repo *note.Repository
}

// ListOptions controls which notes to return.
//
// If EntityType is empty, all notes across all entities are returned.
// Otherwise EntityUUID must be set and only notes for that entity are returned.
//
// Before and After optionally filter by note date (inclusive).
type ListOptions struct {
	EntityType note.EntityType
	EntityUUID string
	Before     time.Time
	After      time.Time
}

func (l *ListNotes) Execute(opts ListOptions) ([]note.Note, error) {
	var notes []note.Note
	var err error
	if opts.EntityType == "" {
		notes, err = l.Repo.ListAll()
	} else {
		notes, err = l.Repo.List(opts.EntityType, opts.EntityUUID)
	}
	if err != nil {
		return nil, err
	}

	if !opts.Before.IsZero() || !opts.After.IsZero() {
		filtered := notes[:0]
		for _, n := range notes {
			if !opts.After.IsZero() && n.Date.Before(opts.After) {
				continue
			}
			if !opts.Before.IsZero() && n.Date.After(opts.Before) {
				continue
			}
			filtered = append(filtered, n)
		}
		notes = filtered
	}

	return notes, nil
}
