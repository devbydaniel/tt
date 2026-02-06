package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type DeleteTasks struct {
	Repo          *task.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (d *DeleteTasks) Execute(ids []int64) (deleted []task.Task, err error) {
	err = d.DB.RunInTx(func() error {
		for _, id := range ids {
			t, err := d.Repo.GetByID(id)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return task.ErrTaskNotFound
				}
				return err
			}

			// If it's a project, delete all children first (each emits sync event)
			if t.IsProject() {
				children, err := d.Repo.ListAllChildren(id)
				if err != nil {
					return err
				}
				for _, child := range children {
					if err := d.Repo.Delete(child.ID); err != nil {
						return err
					}
					// Emit sync event for child
					if d.SyncPersister != nil {
						childCopy := child // avoid capturing loop variable
						if _, err := d.SyncPersister.Execute(&synceventusecases.PersistOptions{
							ClientID:   d.ClientID,
							EventType:  syncevent.EventTypeDeleted,
							Task:       &childCopy,
							EntityUUID: child.UUID,
						}); err != nil {
							return err
						}
					}
					deleted = append(deleted, child)
				}
			}

			// Delete the task itself
			if err := d.Repo.Delete(id); err != nil {
				return err
			}

			// Emit sync event with task data captured before deletion
			if d.SyncPersister != nil {
				if _, err := d.SyncPersister.Execute(&synceventusecases.PersistOptions{
					ClientID:   d.ClientID,
					EventType:  syncevent.EventTypeDeleted,
					Task:       t,
					EntityUUID: t.UUID,
				}); err != nil {
					return err
				}
			}

			deleted = append(deleted, *t)
		}

		return nil
	})
	return
}
