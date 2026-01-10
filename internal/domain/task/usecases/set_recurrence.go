package usecases

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetRecurrence struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
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
