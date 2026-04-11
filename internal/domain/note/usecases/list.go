package usecases

import (
	"time"

	"github.com/devbydaniel/tt/internal/domain/note"
)

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
			date := n.Date.Truncate(24 * time.Hour)
			if !opts.After.IsZero() && date.Before(opts.After) {
				continue
			}
			if !opts.Before.IsZero() && date.After(opts.Before) {
				continue
			}
			filtered = append(filtered, n)
		}
		notes = filtered
	}

	return notes, nil
}
