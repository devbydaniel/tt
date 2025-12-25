package output

import (
	"fmt"
	"io"

	"github.com/devbydaniel/t/internal/domain/task"
)

type Formatter struct {
	w io.Writer
}

func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

func (f *Formatter) TaskCreated(t *task.Task) {
	fmt.Fprintf(f.w, "Created task #%d: %s\n", t.ID, t.Title)
}

func (f *Formatter) TaskList(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No tasks")
		return
	}

	for _, t := range tasks {
		fmt.Fprintf(f.w, "%d  %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) TasksCompleted(tasks []task.Task) {
	for _, t := range tasks {
		fmt.Fprintf(f.w, "Completed #%d: %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) Logbook(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No completed tasks")
		return
	}

	for _, t := range tasks {
		completedAt := ""
		if t.CompletedAt != nil {
			completedAt = t.CompletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(f.w, "%d  %s  %s\n", t.ID, completedAt, t.Title)
	}
}
