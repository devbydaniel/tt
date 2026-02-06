package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// TaskListerByArea lists all tasks in an area (for cascaded deletion)
type TaskListerByArea interface {
	ListAllByArea(areaID int64) ([]task.Task, error)
}

// TaskDeleter deletes tasks (handles children + sync events)
type TaskDeleter interface {
	Execute(ids []int64) ([]task.Task, error)
}

type DeleteArea struct {
	Repo          *area.Repository
	TaskLister    TaskListerByArea
	TaskDeleter   TaskDeleter
	SyncPersister SyncEventPersister
	ClientID      string
}

func (d *DeleteArea) Execute(name string) (*area.Area, error) {
	a, err := d.Repo.GetByName(name)
	if err != nil {
		return nil, err
	}

	// Delete all tasks in this area first (handles children + sync events)
	if d.TaskLister != nil && d.TaskDeleter != nil {
		tasks, err := d.TaskLister.ListAllByArea(a.ID)
		if err != nil {
			return nil, err
		}
		if len(tasks) > 0 {
			// Collect task IDs (projects first so children are handled by DeleteTasks)
			var ids []int64
			for _, t := range tasks {
				if t.IsProject() {
					ids = append(ids, t.ID)
				}
			}
			for _, t := range tasks {
				if !t.IsProject() {
					ids = append(ids, t.ID)
				}
			}
			if _, err := d.TaskDeleter.Execute(ids); err != nil {
				return nil, err
			}
		}
	}

	if err := d.Repo.Delete(a.ID); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if d.SyncPersister != nil {
		_, _ = d.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:   d.ClientID,
			EventType:  syncevent.EventTypeDeleted,
			Area:       a,
			EntityUUID: a.UUID,
		})
	}

	return a, nil
}
