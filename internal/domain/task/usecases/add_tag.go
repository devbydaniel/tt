package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type AddTag struct {
	Repo *task.Repository
}

func (a *AddTag) Execute(id int64, tagName string) (*task.Task, error) {
	// Verify task exists
	if _, err := a.Repo.GetByID(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	if err := a.Repo.AddTag(id, tagName); err != nil {
		return nil, err
	}

	// Reload to get updated tags
	return a.Repo.GetByID(id)
}
