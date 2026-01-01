package usecases

import "github.com/devbydaniel/tt/internal/domain/area"

type ListAreas struct {
	Repo *area.Repository
}

func (l *ListAreas) Execute() ([]area.Area, error) {
	return l.Repo.List()
}
