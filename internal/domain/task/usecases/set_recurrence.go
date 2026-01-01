package usecases

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetRecurrence struct {
	Repo *task.Repository
}

func (s *SetRecurrence) Execute(id int64, recurType, recurRule *string, recurEnd *time.Time) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.RecurType = recurType
	t.RecurRule = recurRule
	t.RecurEnd = recurEnd

	// If setting recurrence, unpause
	if recurType != nil {
		t.RecurPaused = false
	}

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
