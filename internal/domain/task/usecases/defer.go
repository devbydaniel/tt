package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeferTask struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (d *DeferTask) Execute(id int64) (result *task.Task, err error) {
	err = d.DB.RunInTx(func() error {
		t, err := d.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		t.State = task.StateSomeday
		t.PlannedDate = nil // clear planned date when deferring

		if err := d.Repo.Update(t); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if d.SyncPersister != nil {
			if _, err := d.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  d.ClientID,
				EventType: syncevent.EventTypeUpdated,
				Task:      t,
			}); err != nil {
				return err
			}
		}

		result = t
		return nil
	})
	return
}
