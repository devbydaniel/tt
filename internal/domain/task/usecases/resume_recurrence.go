package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type ResumeRecurrence struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (r *ResumeRecurrence) Execute(id int64) (result *task.Task, err error) {
	err = r.DB.RunInTx(func() error {
		t, err := r.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		t.RecurPaused = false

		if err := r.Repo.Update(t); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if r.SyncPersister != nil {
			if _, err := r.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  r.ClientID,
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
