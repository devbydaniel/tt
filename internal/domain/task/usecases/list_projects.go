package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

type ListProjects struct {
	Repo *task.Repository
}

func (l *ListProjects) Execute() ([]task.Task, error) {
	return l.Repo.List(&task.ListFilter{
		TaskType: task.TaskTypeProject,
		State:    task.StateActive,
	})
}
