package usecases

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetRecurrenceEnd struct {
	Repo *task.Repository
}

func (s *SetRecurrenceEnd) Execute(id int64, endDate *time.Time) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.RecurEnd = endDate

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
