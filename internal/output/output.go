package output

import (
	"fmt"
	"io"

	"github.com/devbydaniel/t/internal/domain/area"
	"github.com/devbydaniel/t/internal/domain/project"
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

func (f *Formatter) AreaCreated(a *area.Area) {
	fmt.Fprintf(f.w, "Created area: %s\n", a.Name)
}

func (f *Formatter) AreaList(areas []area.Area) {
	if len(areas) == 0 {
		fmt.Fprintln(f.w, "No areas")
		return
	}

	for _, a := range areas {
		fmt.Fprintln(f.w, a.Name)
	}
}

func (f *Formatter) AreaDeleted(a *area.Area) {
	fmt.Fprintf(f.w, "Deleted area: %s\n", a.Name)
}

func (f *Formatter) ProjectCreated(p *project.Project) {
	fmt.Fprintf(f.w, "Created project: %s\n", p.Name)
}

func (f *Formatter) ProjectList(projects []project.Project) {
	if len(projects) == 0 {
		fmt.Fprintln(f.w, "No projects")
		return
	}

	for _, p := range projects {
		fmt.Fprintln(f.w, p.Name)
	}
}

func (f *Formatter) ProjectDeleted(p *project.Project) {
	fmt.Fprintf(f.w, "Deleted project: %s\n", p.Name)
}
