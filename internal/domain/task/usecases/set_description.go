package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTaskDescription struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
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
