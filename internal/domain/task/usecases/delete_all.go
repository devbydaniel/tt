package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

// DeleteAllTasks removes all tasks from the database.
type DeleteAllTasks struct {
	Repo *task.Repository
}

func (d *DeleteAllTasks) Execute() (int64, error) {
	return d.Repo.DeleteAll()
}
