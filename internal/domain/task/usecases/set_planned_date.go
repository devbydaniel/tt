package usecases

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetPlannedDate struct {
	Repo *task.Repository
}

func (s *SetPlannedDate) Execute(id int64, date *time.Time) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.PlannedDate = date

	// Setting a planned date activates a someday task
	if date != nil && t.State == task.StateSomeday {
		t.State = task.StateActive
	}

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
