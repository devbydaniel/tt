package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTaskTitle struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
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

	// Emit sync event if sync is enabled
	if s.SyncPersister != nil {
		s.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  s.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Task:      t,
		})
	}

	return t, nil
}
