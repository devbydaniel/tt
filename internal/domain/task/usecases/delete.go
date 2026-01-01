package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeleteTasks struct {
	Repo *task.Repository
}

func (d *DeleteTasks) Execute(ids []int64) ([]task.Task, error) {
	var deleted []task.Task

	for _, id := range ids {
		t, err := d.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return deleted, task.ErrTaskNotFound
			}
			return deleted, err
		}
		if err := d.Repo.Delete(id); err != nil {
			return deleted, err
		}
		deleted = append(deleted, *t)
	}

	return deleted, nil
}
