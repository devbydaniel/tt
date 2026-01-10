package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type GetAreaByID struct {
	Repo *area.Repository
}

func (g *GetAreaByID) Execute(id int64) (*area.Area, error) {
	return g.Repo.GetByID(id)
}
