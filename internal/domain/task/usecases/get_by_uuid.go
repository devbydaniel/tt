package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/task"
)

type GetTaskByUUID struct {
	Repo *task.Repository
}

func (g *GetTaskByUUID) Execute(uuid string) (*task.Task, error) {
	return g.Repo.GetByUUID(uuid)
}
