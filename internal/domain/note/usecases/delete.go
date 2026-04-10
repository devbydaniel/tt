package usecases

import "github.com/devbydaniel/tt/internal/domain/note"

// DeleteNote removes a note file from disk.
type DeleteNote struct {
	Repo *note.Repository
}

func (d *DeleteNote) Execute(path string) error {
	return d.Repo.Delete(path)
}
