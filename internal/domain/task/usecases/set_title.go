package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTaskTitle struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (s *SetTaskTitle) Execute(id int64, title string) (result *task.Task, err error) {
	err = s.DB.RunInTx(func() error {
		t, err := s.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		t.Title = title

		if err := s.Repo.Update(t); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if s.SyncPersister != nil {
			if _, err := s.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  s.ClientID,
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
