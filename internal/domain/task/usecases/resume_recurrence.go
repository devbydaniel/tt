package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type ResumeRecurrence struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (r *ResumeRecurrence) Execute(id int64) (*task.Task, error) {
	t, err := r.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.RecurPaused = false

	if err := r.Repo.Update(t); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if r.SyncPersister != nil {
		r.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  r.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Task:      t,
		})
	}

	return t, nil
}
