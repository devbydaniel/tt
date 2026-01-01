package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

type GetProjectByName struct {
	Repo *task.Repository
}

func (g *GetProjectByName) Execute(name string) (*task.Task, error) {
	return g.Repo.GetByName(name, task.TaskTypeProject)
}
