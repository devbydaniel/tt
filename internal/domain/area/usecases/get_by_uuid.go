package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type GetAreaByUUID struct {
	Repo *area.Repository
}

func (g *GetAreaByUUID) Execute(uuid string) (*area.Area, error) {
	return g.Repo.GetByUUID(uuid)
}
