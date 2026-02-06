package usecases

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeleteTag struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (d *DeleteTag) Execute(tagName string) (count int64, err error) {
	err = d.DB.RunInTx(func() error {
		// Get task IDs that have this tag (for sync events)
		taskIDs, err := d.Repo.GetTaskIDsByTag(tagName)
		if err != nil {
			return err
		}

		// Delete the tag from all tasks
		count, err = d.Repo.DeleteTag(tagName)
		if err != nil {
			return err
		}

		if count == 0 {
			return task.ErrTagNotFound
		}

		// Emit sync events for each affected task
		if d.SyncPersister != nil {
			for _, id := range taskIDs {
				t, err := d.Repo.GetByID(id)
				if err != nil {
					continue // task may have been deleted
				}
				if _, err := d.SyncPersister.Execute(&synceventusecases.PersistOptions{
					ClientID:  d.ClientID,
					EventType: syncevent.EventTypeUpdated,
					Task:      t,
				}); err != nil {
					return err
				}
			}
		}

		return nil
	})
	return
}
