package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type DeleteArea struct {
	Repo *area.Repository
}

func (d *DeleteArea) Execute(name string) (*area.Area, error) {
	a, err := d.Repo.GetByName(name)
	if err != nil {
		return nil, err
	}

	if err := d.Repo.Delete(a.ID); err != nil {
		return nil, err
	}

	return a, nil
}
