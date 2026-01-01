package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// AreaLookupForSetArea is what this use case needs from the area domain
type AreaLookupForSetArea interface {
	Execute(name string) (*area.Area, error)
}

type SetTaskArea struct {
	Repo       *task.Repository
	AreaLookup AreaLookupForSetArea
}

func (s *SetTaskArea) Execute(id int64, areaName string) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	if areaName == "" {
		t.AreaID = nil
	} else {
		a, err := s.AreaLookup.Execute(areaName)
		if err != nil {
			return nil, err
		}
		t.AreaID = &a.ID
		// Clear parent when setting area (mutual exclusivity)
		t.ParentID = nil
	}

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
