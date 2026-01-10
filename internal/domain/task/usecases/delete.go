package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeleteTasks struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (d *DeleteTasks) Execute(ids []int64) ([]task.Task, error) {
	var deleted []task.Task

	for _, id := range ids {
		t, err := d.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return deleted, task.ErrTaskNotFound
			}
			return deleted, err
		}
		if err := d.Repo.Delete(id); err != nil {
			return deleted, err
		}

		// Emit sync event with task data captured before deletion
		if d.SyncPersister != nil {
			d.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:   d.ClientID,
				EventType:  syncevent.EventTypeDeleted,
				Task:       t,
				EntityUUID: t.UUID,
			})
		}

		deleted = append(deleted, *t)
	}

	return deleted, nil
}
