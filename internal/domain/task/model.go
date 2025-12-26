package task

import "time"

type Task struct {
	ID          int64
	UUID        string
	Title       string
	ProjectID   *int64
	AreaID      *int64
	PlannedDate *time.Time
	DueDate     *time.Time
	State       string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time

	// Recurrence fields
	RecurType     *string    // "fixed" or "relative"
	RecurRule     *string    // JSON rule: {"interval":1,"unit":"week",...}
	RecurEnd      *time.Time // optional end date
	RecurPaused   bool       // true = paused
	RecurParentID *int64     // links to original recurring task
}

// Recurrence type constants
const (
	RecurTypeFixed    = "fixed"
	RecurTypeRelative = "relative"
)

const (
	StatusTodo = "todo"
	StatusDone = "done"

	StateActive  = "active"
	StateSomeday = "someday"
)
