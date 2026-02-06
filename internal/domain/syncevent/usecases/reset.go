package usecases

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// ResetResult contains the result of a sync reset.
type ResetResult struct {
	DeletedEvents     int64
	RegeneratedEvents int
}

// ResetSyncEvents clears sync events and resets the cursor.
// Used server-side where we only need to clear the event log.
type ResetSyncEvents struct {
	Repo *syncevent.Repository
}

// Execute deletes all sync events, resets the cursor, and clears the pending resolution queue.
func (r *ResetSyncEvents) Execute() (*ResetResult, error) {
	count, err := r.Repo.DeleteAll()
	if err != nil {
		return nil, err
	}

	if err := r.Repo.SetSyncState(SyncStateServerCursor, "0"); err != nil {
		return nil, err
	}

	if err := r.Repo.DeleteAllPending(); err != nil {
		return nil, err
	}

	return &ResetResult{DeletedEvents: count}, nil
}

// SyncPersister creates sync events for entities.
type SyncPersister interface {
	Execute(opts *PersistOptions) (*syncevent.SyncEvent, error)
}

// AllTasksLister lists all tasks (both todo and done).
type AllTasksLister interface {
	Execute() ([]task.Task, error)
}

// AllAreasLister lists all areas.
type AllAreasLister interface {
	Execute() ([]area.Area, error)
}

// ResetSync performs a full sync reset:
// 1. Resets the server (clears events + materialized data)
// 2. Clears local sync events and cursor
// 3. Regenerates sync events for all local tasks and areas
type ResetSync struct {
	Repo          *syncevent.Repository
	Client        *syncevent.Client // nil if server not configured
	TaskLister    AllTasksLister
	AreaLister    AllAreasLister
	SyncPersister SyncPersister
	ClientID      string
	DB            *database.DB
}

// Execute performs the full sync reset.
func (r *ResetSync) Execute() (result *ResetResult, err error) {
	// Step 1: Reset the server (if configured) — outside transaction (network call)
	if r.Client != nil {
		if err := r.Client.Reset(); err != nil {
			return nil, err
		}
	}

	err = r.DB.RunInTx(func() error {
		result = &ResetResult{}

		// Step 2: Clear local sync events, cursor, and pending resolution queue
		deleted, err := r.Repo.DeleteAll()
		if err != nil {
			return err
		}
		result.DeletedEvents = deleted

		if err := r.Repo.SetSyncState(SyncStateServerCursor, "0"); err != nil {
			return err
		}

		if err := r.Repo.DeleteAllPending(); err != nil {
			return err
		}

		// Step 3: Regenerate sync events for all local areas
		areas, err := r.AreaLister.Execute()
		if err != nil {
			return err
		}

		for i := range areas {
			_, err := r.SyncPersister.Execute(&PersistOptions{
				ClientID:  r.ClientID,
				EventType: syncevent.EventTypeCreated,
				Area:      &areas[i],
			})
			if err != nil {
				return err
			}
			result.RegeneratedEvents++
		}

		// Step 4: Regenerate sync events for all local tasks
		tasks, err := r.TaskLister.Execute()
		if err != nil {
			return err
		}

		for i := range tasks {
			eventType := syncevent.EventTypeCreated
			if tasks[i].Status == task.StatusDone {
				eventType = syncevent.EventTypeCompleted
			}
			_, err := r.SyncPersister.Execute(&PersistOptions{
				ClientID:  r.ClientID,
				EventType: eventType,
				Task:      &tasks[i],
			})
			if err != nil {
				return err
			}
			result.RegeneratedEvents++
		}

		return nil
	})
	return
}
