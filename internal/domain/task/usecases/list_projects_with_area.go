package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

type ListProjectsWithArea struct {
	Repo *task.Repository
}

// Execute returns all active projects with their area names populated
// The AreaName field is populated via LEFT JOIN in the repository
func (l *ListProjectsWithArea) Execute() ([]task.Task, error) {
	return l.Repo.List(&task.ListFilter{
		TaskType: task.TaskTypeProject,
		State:    task.StateActive,
	})
}
