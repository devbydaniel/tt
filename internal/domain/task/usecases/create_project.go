package usecases

import (
	"time"

	"github.com/google/uuid"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// AreaLookupForCreateProject is what this use case needs from the area domain
type AreaLookupForCreateProject interface {
	Execute(name string) (*area.Area, error)
}

// CreateProjectOptions contains options for creating a project
type CreateProjectOptions struct {
	AreaName    string
	Description string
	PlannedDate *time.Time
	DueDate     *time.Time
	Someday     bool
}

type CreateProject struct {
	Repo          *task.Repository
	AreaLookup    AreaLookupForCreateProject
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (c *CreateProject) Execute(name string, opts *CreateProjectOptions) (result *task.Task, err error) {
	err = c.DB.RunInTx(func() error {
		state := task.StateActive
		if opts != nil && opts.Someday {
			state = task.StateSomeday
		}

		p := &task.Task{
			UUID:      uuid.New().String(),
			Title:     name,
			TaskType:  task.TaskTypeProject,
			State:     state,
			Status:    task.StatusTodo,
			CreatedAt: time.Now(),
		}

		if opts != nil {
			if opts.Description != "" {
				p.Description = &opts.Description
			}
			p.PlannedDate = opts.PlannedDate
			p.DueDate = opts.DueDate

			if opts.AreaName != "" {
				a, err := c.AreaLookup.Execute(opts.AreaName)
				if err != nil {
					return err
				}
				p.AreaID = &a.ID
			}
		}

		if err := c.Repo.Create(p); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if c.SyncPersister != nil {
			if _, err := c.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  c.ClientID,
				EventType: syncevent.EventTypeCreated,
				Task:      p,
			}); err != nil {
				return err
			}
		}

		result = p
		return nil
	})
	return
}
