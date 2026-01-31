package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type PauseRecurrence struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (p *PauseRecurrence) Execute(id int64) (*task.Task, error) {
	t, err := p.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.RecurPaused = true

	if err := p.Repo.Update(t); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if p.SyncPersister != nil {
		_, _ = p.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  p.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Task:      t,
		})
	}

	return t, nil
}
