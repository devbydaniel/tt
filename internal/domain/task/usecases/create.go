package usecases

import (
	"time"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/google/uuid"
)

// ProjectLookup is what this use case needs to look up projects (which are now tasks)
type ProjectLookup interface {
	Execute(name string) (*task.Task, error)
}

// AreaLookup is what this use case needs from the area domain
type AreaLookup interface {
	Execute(name string) (*area.Area, error)
}

type CreateTask struct {
	Repo          *task.Repository
	ProjectLookup ProjectLookup
	AreaLookup    AreaLookup
}

func (c *CreateTask) Execute(title string, opts *task.CreateOptions) (*task.Task, error) {
	t := &task.Task{
		UUID:      uuid.New().String(),
		Title:     title,
		TaskType:  task.TaskTypeTask,
		State:     task.StateActive,
		Status:    task.StatusTodo,
		CreatedAt: time.Now(),
	}

	if opts != nil {
		if opts.ProjectName != "" {
			p, err := c.ProjectLookup.Execute(opts.ProjectName)
			if err != nil {
				return nil, err
			}
			t.ParentID = &p.ID
		}
		if opts.AreaName != "" {
			a, err := c.AreaLookup.Execute(opts.AreaName)
			if err != nil {
				return nil, err
			}
			t.AreaID = &a.ID
		}
		if opts.Description != "" {
			t.Description = &opts.Description
		}
		t.PlannedDate = opts.PlannedDate
		t.DueDate = opts.DueDate

		// Recurrence fields
		t.RecurType = opts.RecurType
		t.RecurRule = opts.RecurRule
		t.RecurEnd = opts.RecurEnd
		t.RecurParentID = opts.RecurParentID

		if opts.Someday {
			if opts.PlannedDate == nil && opts.DueDate == nil {
				t.State = task.StateSomeday
			}
		}
	}

	if err := c.Repo.Create(t); err != nil {
		return nil, err
	}

	// Save tags if provided
	if opts != nil && len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			if err := c.Repo.AddTag(t.ID, tag); err != nil {
				return nil, err
			}
		}
		t.Tags = opts.Tags
	}

	return t, nil
}
