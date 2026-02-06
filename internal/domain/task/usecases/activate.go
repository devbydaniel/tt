package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type ActivateTask struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (a *ActivateTask) Execute(id int64) (result *task.Task, err error) {
	err = a.DB.RunInTx(func() error {
		t, err := a.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		t.State = task.StateActive

		if err := a.Repo.Update(t); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if a.SyncPersister != nil {
			if _, err := a.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  a.ClientID,
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
