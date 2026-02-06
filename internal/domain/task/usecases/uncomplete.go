package usecases

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type UncompleteTasks struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (u *UncompleteTasks) Execute(ids []int64) (tasks []task.Task, err error) {
	err = u.DB.RunInTx(func() error {
		for _, id := range ids {
			if err := u.Repo.Uncomplete(id); err != nil {
				return err
			}
			t, err := u.Repo.GetByID(id)
			if err != nil {
				return err
			}

			// Emit sync event if sync is enabled
			if u.SyncPersister != nil {
				if _, err := u.SyncPersister.Execute(&synceventusecases.PersistOptions{
					ClientID:  u.ClientID,
					EventType: syncevent.EventTypeUpdated,
					Task:      t,
				}); err != nil {
					return err
				}
			}

			tasks = append(tasks, *t)
		}

		return nil
	})
	return
}
