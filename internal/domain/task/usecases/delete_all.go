package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// DeleteAllTasks removes all tasks from the database.
type DeleteAllTasks struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

// Execute deletes all tasks. If skipSyncEvents is true, no sync events are
// emitted (used during server reset where all sync state is being cleared).
func (d *DeleteAllTasks) Execute(skipSyncEvents ...bool) (int64, error) {
	skip := len(skipSyncEvents) > 0 && skipSyncEvents[0]

	// Emit sync events for each task before deleting
	if !skip && d.SyncPersister != nil {
		tasks, err := d.Repo.ListAll()
		if err != nil {
			return 0, err
		}
		for i := range tasks {
			_, _ = d.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:   d.ClientID,
				EventType:  syncevent.EventTypeDeleted,
				Task:       &tasks[i],
				EntityUUID: tasks[i].UUID,
			})
		}
	}

	return d.Repo.DeleteAll()
}
