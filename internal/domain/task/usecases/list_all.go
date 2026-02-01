package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

// ListAllTasks returns all tasks (both todo and done) without filters.
type ListAllTasks struct {
	Repo *task.Repository
}

func (l *ListAllTasks) Execute() ([]task.Task, error) {
	return l.Repo.ListAll()
}
