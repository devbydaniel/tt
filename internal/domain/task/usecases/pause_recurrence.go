package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type PauseRecurrence struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (p *PauseRecurrence) Execute(id int64) (result *task.Task, err error) {
	err = p.DB.RunInTx(func() error {
		t, err := p.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		t.RecurPaused = true

		if err := p.Repo.Update(t); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if p.SyncPersister != nil {
			if _, err := p.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  p.ClientID,
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
