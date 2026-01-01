package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type PauseRecurrence struct {
	Repo *task.Repository
}

func (p *PauseRecurrence) Execute(id int64) (*task.Task, error) {
	t, err := p.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.RecurPaused = true

	if err := p.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
