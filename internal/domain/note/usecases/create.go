package usecases

import "github.com/devbydaniel/tt/internal/domain/note"

// CreateNote writes a new markdown note for the given entity.
type CreateNote struct {
	Repo *note.Repository
}

// CreateOptions describes the note to create.
type CreateOptions struct {
	EntityType note.EntityType
	EntityUUID string
	Title      string
	Body       string // optional; if empty, a small header is generated
}

func (c *CreateNote) Execute(opts CreateOptions) (*note.Note, error) {
	return c.Repo.Create(opts.EntityType, opts.EntityUUID, opts.Title, opts.Body)
}
