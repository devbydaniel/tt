package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type CreateArea struct {
	Repo *area.Repository
}

func (c *CreateArea) Execute(name string) (*area.Area, error) {
	a := &area.Area{
		Name: name,
	}

	if err := c.Repo.Create(a); err != nil {
		return nil, err
	}

	return a, nil
}
