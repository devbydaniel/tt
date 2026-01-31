package usecases

import (
	"errors"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// ProjectLookupForList is what this use case needs to look up projects (which are now tasks)
type ProjectLookupForList interface {
	Execute(name string) (*task.Task, error)
}

// AreaLookupForList is what this use case needs from the area domain
type AreaLookupForList interface {
	Execute(name string) (*area.Area, error)
}

type ListTasks struct {
	Repo          *task.Repository
	ProjectLookup ProjectLookupForList
	AreaLookup    AreaLookupForList
}

func (l *ListTasks) Execute(opts *task.ListOptions) ([]task.Task, error) {
	filter := &task.ListFilter{}

	if opts != nil {
		// Allow explicit task type filter if specified
		if opts.TaskType != "" {
			filter.TaskType = opts.TaskType
		}

		if opts.ProjectName != "" {
			p, err := l.ProjectLookup.Execute(opts.ProjectName)
			if err != nil {
				return nil, err
			}
			filter.ParentID = &p.ID
		}
		if opts.AreaName != "" {
			a, err := l.AreaLookup.Execute(opts.AreaName)
			if err != nil {
				return nil, err
			}
			filter.AreaID = &a.ID
		}
		if opts.TagName != "" {
			filter.TagName = opts.TagName
		}
		if len(opts.TagNames) > 0 {
			filter.TagNames = opts.TagNames
		}
		if opts.Search != "" {
			filter.Search = opts.Search
		}
		if len(opts.Sort) > 0 {
			filter.Sort = opts.Sort
		}

		// Explicit state filter (takes precedence over schedule-based state)
		if opts.State != "" {
			filter.State = opts.State
		}

		switch opts.Schedule {
		case "today":
			filter.Today = true
		case "upcoming":
			filter.Upcoming = true
			// Default sort by date ascending (nearest first) if no explicit sort
			if len(filter.Sort) == 0 {
				filter.Sort = []task.SortOption{{Field: task.SortByDate, Direction: task.SortAsc}}
			}
		case "anytime":
			filter.Anytime = true
			filter.TaskType = task.TaskTypeTask // Only show tasks, not projects
		case "inbox":
			filter.Inbox = true
		case "someday":
			// Only set state if not explicitly overridden
			if opts.State == "" {
				filter.State = task.StateSomeday
			}
		}

		// NOT filters for bulk edit (AND logic - all must be satisfied)
		for _, projectName := range opts.NotProjectNames {
			p, err := l.ProjectLookup.Execute(projectName)
			if err != nil {
				// If project not found, the NOT condition is vacuously satisfied
				// (no tasks belong to a non-existent project) - just skip this filter
				if !errors.Is(err, task.ErrTaskNotFound) {
					return nil, err
				}
			} else {
				filter.NotParentIDs = append(filter.NotParentIDs, p.ID)
			}
		}
		for _, areaName := range opts.NotAreaNames {
			a, err := l.AreaLookup.Execute(areaName)
			if err != nil {
				// If area not found, skip this filter
				if !errors.Is(err, area.ErrAreaNotFound) {
					return nil, err
				}
			} else {
				filter.NotAreaIDs = append(filter.NotAreaIDs, a.ID)
			}
		}
		if len(opts.NotTagNames) > 0 {
			filter.NotTagNames = opts.NotTagNames
		}
	}

	return l.Repo.List(filter)
}
