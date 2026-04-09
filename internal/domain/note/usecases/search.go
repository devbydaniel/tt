package usecases

import "github.com/devbydaniel/tt/internal/domain/note"

// SearchNotes performs a case-insensitive substring search across notes.
type SearchNotes struct {
	Repo *note.Repository
}

// SearchOptions controls the search scope.
//
// If EntityType is empty, the entire notes tree is scanned. Otherwise only
// notes for the given entity are scanned.
type SearchOptions struct {
	EntityType note.EntityType
	EntityUUID string
	Query      string
}

func (s *SearchNotes) Execute(opts SearchOptions) ([]note.SearchMatch, error) {
	return s.Repo.Search(opts.EntityType, opts.EntityUUID, opts.Query)
}
