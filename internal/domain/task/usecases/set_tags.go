package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTags struct {
	Repo *task.Repository
}

func (s *SetTags) Execute(id int64, tags []string) (*task.Task, error) {
	// Verify task exists
	if _, err := s.Repo.GetByID(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	if err := s.Repo.SetTags(id, tags); err != nil {
		return nil, err
	}

	// Reload to get updated task with tags
	return s.Repo.GetByID(id)
}
