package usecases

import "github.com/devbydaniel/tt/internal/domain/note"

// ListNotes returns notes for a given entity, or all notes if EntityType is empty.
type ListNotes struct {
	Repo *note.Repository
}

// ListOptions controls which notes to return.
//
// If EntityType is empty, all notes across all entities are returned.
// Otherwise EntityUUID must be set and only notes for that entity are returned.
type ListOptions struct {
	EntityType note.EntityType
	EntityUUID string
}

func (l *ListNotes) Execute(opts ListOptions) ([]note.Note, error) {
	if opts.EntityType == "" {
		return l.Repo.ListAll()
	}
	return l.Repo.List(opts.EntityType, opts.EntityUUID)
}
