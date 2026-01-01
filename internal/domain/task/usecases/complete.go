package usecases

import (
	"time"

	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/recurparse"
	"github.com/google/uuid"
)

type CompleteTasks struct {
	Repo *task.Repository
}

func (c *CompleteTasks) Execute(ids []int64) ([]task.CompleteResult, error) {
	completedAt := time.Now()
	var results []task.CompleteResult

	for _, id := range ids {
		// Get the task first to check if it's a project
		t, err := c.Repo.GetByID(id)
		if err != nil {
			return results, err
		}

		// If it's a project, complete it along with all its children
		if t.IsProject() {
			if err := c.Repo.CompleteWithChildren(id, completedAt); err != nil {
				return results, err
			}
		} else {
			if err := c.Repo.Complete(id, completedAt); err != nil {
				return results, err
			}
		}

		// Refresh the task to get updated status
		t, err = c.Repo.GetByID(id)
		if err != nil {
			return results, err
		}

		result := task.CompleteResult{Completed: *t}

		// Check if task should regenerate (not for projects)
		if !t.IsProject() && t.RecurType != nil && t.RecurRule != nil && !t.RecurPaused {
			nextTask := c.regenerateTask(t, completedAt)
			if nextTask != nil {
				result.NextTask = nextTask
			}
		}

		results = append(results, result)
	}

	return results, nil
}

func (c *CompleteTasks) regenerateTask(t *task.Task, completedAt time.Time) *task.Task {
	// Check if past end date
	if t.RecurEnd != nil && time.Now().After(*t.RecurEnd) {
		return nil
	}

	// Parse the recurrence rule
	rule, err := recurparse.FromJSON(*t.RecurRule)
	if err != nil {
		return nil
	}

	// Calculate next occurrence
	recurrenceType := recurparse.TypeFixed
	if *t.RecurType == task.RecurTypeRelative {
		recurrenceType = recurparse.TypeRelative
	}

	var fromDate time.Time
	if recurrenceType == recurparse.TypeRelative {
		fromDate = completedAt
	} else {
		fromDate = time.Now()
	}
	nextDate := recurparse.NextOccurrence(rule, recurrenceType, fromDate)

	// Determine which date field to set based on original task
	var plannedDate, dueDate *time.Time
	if t.DueDate != nil {
		dueDate = &nextDate
	} else {
		plannedDate = &nextDate
	}

	// Determine the parent ID for linking
	parentID := t.RecurParentID
	if parentID == nil {
		parentID = &t.ID
	}

	// Create the next task
	nextTask := &task.Task{
		UUID:          uuid.New().String(),
		Title:         t.Title,
		Description:   t.Description,
		TaskType:      task.TaskTypeTask,
		ParentID:      t.ParentID,
		AreaID:        t.AreaID,
		PlannedDate:   plannedDate,
		DueDate:       dueDate,
		State:         task.StateActive,
		Status:        task.StatusTodo,
		CreatedAt:     time.Now(),
		RecurType:     t.RecurType,
		RecurRule:     t.RecurRule,
		RecurEnd:      t.RecurEnd,
		RecurParentID: parentID,
	}

	if err := c.Repo.Create(nextTask); err != nil {
		return nil
	}

	// Copy tags from original task
	if len(t.Tags) > 0 {
		for _, tag := range t.Tags {
			if err := c.Repo.AddTag(nextTask.ID, tag); err != nil {
				return nil
			}
		}
		nextTask.Tags = t.Tags
	}

	return nextTask
}
