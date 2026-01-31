package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type ActivateTask struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (a *ActivateTask) Execute(id int64) (*task.Task, error) {
	t, err := a.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.State = task.StateActive

	if err := a.Repo.Update(t); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if a.SyncPersister != nil {
		_, _ = a.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  a.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Task:      t,
		})
	}

	return t, nil
}
