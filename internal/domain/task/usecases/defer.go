package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeferTask struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (d *DeferTask) Execute(id int64) (*task.Task, error) {
	t, err := d.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	t.State = task.StateSomeday
	t.PlannedDate = nil // clear planned date when deferring

	if err := d.Repo.Update(t); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if d.SyncPersister != nil {
		_, _ = d.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  d.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Task:      t,
		})
	}

	return t, nil
}
