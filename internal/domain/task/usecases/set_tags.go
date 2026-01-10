package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTags struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
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
	t, err := s.Repo.GetByID(id)
	if err != nil {
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
