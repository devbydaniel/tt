package task

import "time"

type Task struct {
	ID          int64
	UUID        string
	Title       string
	ProjectID   *int64
	AreaID      *int64
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

const (
	StatusTodo = "todo"
	StatusDone = "done"
)
