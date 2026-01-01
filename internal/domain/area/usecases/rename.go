package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type RenameArea struct {
	Repo *area.Repository
}

func (r *RenameArea) Execute(oldName, newName string) (*area.Area, error) {
	a, err := r.Repo.GetByName(oldName)
	if err != nil {
		return nil, err
	}

	a.Name = newName
	if err := r.Repo.Update(a); err != nil {
		return nil, err
	}

	return a, nil
}
