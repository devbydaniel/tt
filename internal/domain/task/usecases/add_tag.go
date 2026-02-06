package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type AddTag struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (a *AddTag) Execute(id int64, tagName string) (result *task.Task, err error) {
	err = a.DB.RunInTx(func() error {
		// Verify task exists
		if _, err := a.Repo.GetByID(id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		if err := a.Repo.AddTag(id, tagName); err != nil {
			return err
		}

		// Reload to get updated tags
		t, err := a.Repo.GetByID(id)
		if err != nil {
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
