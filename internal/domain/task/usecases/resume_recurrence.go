package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type ResumeRecurrence struct {
	Repo *task.Repository
}

func (r *ResumeRecurrence) Execute(id int64) (*task.Task, error) {
	t, err := r.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.RecurPaused = false

	if err := r.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
