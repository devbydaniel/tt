package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

// DeleteAllAreas removes all areas from the database.
type DeleteAllAreas struct {
	Repo *area.Repository
}

func (d *DeleteAllAreas) Execute() (int64, error) {
	return d.Repo.DeleteAll()
}
