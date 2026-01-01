package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

type GetTask struct {
	Repo *task.Repository
}

func (g *GetTask) Execute(id int64) (*task.Task, error) {
	t, err := g.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}
	return t, nil
}
