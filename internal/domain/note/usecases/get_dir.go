package usecases

import "github.com/devbydaniel/tt/internal/domain/note"

// GetNoteDir returns the filesystem directory for notes belonging to a given entity.
type GetNoteDir struct {
	Repo *note.Repository
}

func (g *GetNoteDir) Execute(entityType note.EntityType, entityUUID string) string {
	return g.Repo.EntityDir(entityType, entityUUID)
}
