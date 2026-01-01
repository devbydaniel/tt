package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

type ListAllProjects struct {
	Repo *task.Repository
}

func (l *ListAllProjects) Execute() ([]task.Task, error) {
	return l.Repo.List(&task.ListFilter{
		TaskType: task.TaskTypeProject,
		// No state filter - returns both active and someday projects
	})
}
