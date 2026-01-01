package usecases

import (
	"time"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/google/uuid"
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
	Repo       *task.Repository
	AreaLookup AreaLookupForCreateProject
}

func (c *CreateProject) Execute(name string, opts *CreateProjectOptions) (*task.Task, error) {
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
				return nil, err
			}
			p.AreaID = &a.ID
		}
	}

	if err := c.Repo.Create(p); err != nil {
		return nil, err
	}

	return p, nil
}
