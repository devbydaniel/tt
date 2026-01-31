package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type UncompleteTasks struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (u *UncompleteTasks) Execute(ids []int64) ([]task.Task, error) {
	var tasks []task.Task

	for _, id := range ids {
		if err := u.Repo.Uncomplete(id); err != nil {
			return tasks, err
		}
		t, err := u.Repo.GetByID(id)
		if err != nil {
			return tasks, err
		}

		// Emit sync event if sync is enabled
		if u.SyncPersister != nil {
			_, _ = u.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  u.ClientID,
				EventType: syncevent.EventTypeUpdated,
				Task:      t,
			})
		}

		tasks = append(tasks, *t)
	}

	return tasks, nil
}
