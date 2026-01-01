package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type GetAreaByName struct {
	Repo *area.Repository
}

func (g *GetAreaByName) Execute(name string) (*area.Area, error) {
	return g.Repo.GetByName(name)
}
