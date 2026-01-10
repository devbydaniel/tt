package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type RemoveTag struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (r *RemoveTag) Execute(id int64, tagName string) (*task.Task, error) {
	// Verify task exists
	if _, err := r.Repo.GetByID(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	if err := r.Repo.RemoveTag(id, tagName); err != nil {
		return nil, err
	}

	// Reload to get updated tags
	t, err := r.Repo.GetByID(id)
	if err != nil {
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
