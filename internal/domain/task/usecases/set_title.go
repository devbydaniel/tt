package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTaskTitle struct {
	Repo *task.Repository
}

func (s *SetTaskTitle) Execute(id int64, title string) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.Title = title

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
