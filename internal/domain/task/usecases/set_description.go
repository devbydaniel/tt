package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTaskDescription struct {
	Repo *task.Repository
}

func (s *SetTaskDescription) Execute(id int64, description *string) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.Description = description

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
