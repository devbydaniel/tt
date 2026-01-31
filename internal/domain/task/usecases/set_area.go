package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// AreaLookupForSetArea is what this use case needs from the area domain
type AreaLookupForSetArea interface {
	Execute(name string) (*area.Area, error)
}

type SetTaskArea struct {
	Repo          *task.Repository
	AreaLookup    AreaLookupForSetArea
	SyncPersister SyncEventPersister
	ClientID      string
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

	// Emit sync event if sync is enabled
	if s.SyncPersister != nil {
		_, _ = s.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  s.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Task:      t,
		})
	}

	return t, nil
}
