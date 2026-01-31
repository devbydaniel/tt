package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type AddTag struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (a *AddTag) Execute(id int64, tagName string) (*task.Task, error) {
	// Verify task exists
	if _, err := a.Repo.GetByID(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	if err := a.Repo.AddTag(id, tagName); err != nil {
		return nil, err
	}

	// Reload to get updated tags
	t, err := a.Repo.GetByID(id)
	if err != nil {
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
