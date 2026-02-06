package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type SetTags struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (s *SetTags) Execute(id int64, tags []string) (result *task.Task, err error) {
	err = s.DB.RunInTx(func() error {
		// Verify task exists
		if _, err := s.Repo.GetByID(id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		if err := s.Repo.SetTags(id, tags); err != nil {
			return err
		}

		// Reload to get updated task with tags
		t, err := s.Repo.GetByID(id)
		if err != nil {
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
