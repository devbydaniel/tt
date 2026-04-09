package usecases

import (
	"fmt"

	"github.com/devbydaniel/tt/internal/domain/comment"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type TaskLookup interface {
	Execute(id int64) (*task.Task, error)
}

type AddComment struct {
	Repo       *comment.Repository
	TaskLookup TaskLookup
}

type AddOptions struct {
	TaskID int64
	Author string
	Body   string
}

func (a *AddComment) Execute(opts AddOptions) (*comment.Comment, error) {
	if opts.Author == "" {
		return nil, fmt.Errorf("author is required")
	}
	if opts.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	// Verify the task exists
	if _, err := a.TaskLookup.Execute(opts.TaskID); err != nil {
		return nil, fmt.Errorf("task lookup: %w", err)
	}

	c := &comment.Comment{
		TaskID: opts.TaskID,
		Author: opts.Author,
		Body:   opts.Body,
	}

	if err := a.Repo.Create(c); err != nil {
		return nil, err
	}

	return c, nil
}
