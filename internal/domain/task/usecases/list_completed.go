package usecases

import (
	"time"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type ListCompletedTasks struct {
	Repo *task.Repository
}

func (l *ListCompletedTasks) Execute(since *time.Time) ([]task.Task, error) {
	return l.Repo.ListCompleted(since)
}
