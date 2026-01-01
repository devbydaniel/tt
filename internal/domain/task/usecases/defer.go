package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeferTask struct {
	Repo *task.Repository
}

func (d *DeferTask) Execute(id int64) (*task.Task, error) {
	t, err := d.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.State = task.StateSomeday
	t.PlannedDate = nil // clear planned date when deferring

	if err := d.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
