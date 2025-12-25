package task

import "time"

type Task struct {
	ID        int64
	UUID      string
	Title     string
	Status    string
	CreatedAt time.Time
}

const (
	StatusTodo = "todo"
	StatusDone = "done"
)
