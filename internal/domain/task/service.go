package task

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/recurparse"
	"github.com/google/uuid"
)

type Service struct {
	repo           *Repository
	projectService *project.Service
	areaService    *area.Service
}

func NewService(repo *Repository, projectService *project.Service, areaService *area.Service) *Service {
	return &Service{
		repo:           repo,
		projectService: projectService,
		areaService:    areaService,
	}
}

type CreateOptions struct {
	ProjectName string
	AreaName    string
	PlannedDate *time.Time
	DueDate     *time.Time
	Someday     bool     // if true, create in someday state
	Tags        []string // tags to assign

	// Recurrence options
	RecurType     *string    // "fixed" or "relative"
	RecurRule     *string    // JSON rule
	RecurEnd      *time.Time // optional end date
	RecurParentID *int64     // for linking regenerated tasks
}

func (s *Service) Create(title string, opts *CreateOptions) (*Task, error) {
	task := &Task{
		UUID:      uuid.New().String(),
		Title:     title,
		State:     StateActive,
		Status:    StatusTodo,
		CreatedAt: time.Now(),
	}

	if opts != nil {
		if opts.ProjectName != "" {
			p, err := s.projectService.GetByName(opts.ProjectName)
			if err != nil {
				return nil, err
			}
			task.ProjectID = &p.ID
		}
		if opts.AreaName != "" {
			a, err := s.areaService.GetByName(opts.AreaName)
			if err != nil {
				return nil, err
			}
			task.AreaID = &a.ID
		}
		task.PlannedDate = opts.PlannedDate
		task.DueDate = opts.DueDate

		// Recurrence fields
		task.RecurType = opts.RecurType
		task.RecurRule = opts.RecurRule
		task.RecurEnd = opts.RecurEnd
		task.RecurParentID = opts.RecurParentID

		if opts.Someday {
			// If someday is requested but dates are provided, stay active
			// (adding dates to someday → active)
			if opts.PlannedDate == nil && opts.DueDate == nil {
				task.State = StateSomeday
			}
		}
	}

	if err := s.repo.Create(task); err != nil {
		return nil, err
	}

	// Save tags if provided
	if opts != nil && len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			if err := s.repo.AddTag(task.ID, tag); err != nil {
				return nil, err
			}
		}
		task.Tags = opts.Tags
	}

	return task, nil
}

type ListOptions struct {
	ProjectName string
	AreaName    string
	TagName     string // filter by tag
	Today       bool   // show today + overdue
	Upcoming    bool   // show future dates
	Someday     bool   // show someday tasks
	Anytime     bool   // show active tasks with no dates
	Inbox       bool   // show tasks with no project/area/dates
	All         bool   // show all active (no date filter)
}

func (s *Service) List(opts *ListOptions) ([]Task, error) {
	filter := &ListFilter{
		State: StateActive, // default to active tasks
	}

	if opts != nil {
		if opts.ProjectName != "" {
			p, err := s.projectService.GetByName(opts.ProjectName)
			if err != nil {
				return nil, err
			}
			filter.ProjectID = &p.ID
		}
		if opts.AreaName != "" {
			a, err := s.areaService.GetByName(opts.AreaName)
			if err != nil {
				return nil, err
			}
			filter.AreaID = &a.ID
		}
		if opts.TagName != "" {
			filter.TagName = opts.TagName
		}

		if opts.Someday {
			filter.State = StateSomeday
		} else if opts.Today {
			filter.Today = true
		} else if opts.Upcoming {
			filter.Upcoming = true
		} else if opts.Anytime {
			filter.Anytime = true
		} else if opts.Inbox {
			filter.Inbox = true
		}
		// opts.All means no date filter, just state = active (default)
	}

	return s.repo.List(filter)
}

// CompleteResult represents the result of completing a task.
type CompleteResult struct {
	Completed Task
	NextTask  *Task // non-nil if a recurring task was regenerated
}

func (s *Service) Complete(ids []int64) ([]CompleteResult, error) {
	completedAt := time.Now()
	var results []CompleteResult

	for _, id := range ids {
		if err := s.repo.Complete(id, completedAt); err != nil {
			return results, err
		}
		task, err := s.repo.GetByID(id)
		if err != nil {
			return results, err
		}

		result := CompleteResult{Completed: *task}

		// Check if task should regenerate
		if task.RecurType != nil && task.RecurRule != nil && !task.RecurPaused {
			nextTask := s.regenerateTask(task, completedAt)
			if nextTask != nil {
				result.NextTask = nextTask
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// regenerateTask creates the next occurrence of a recurring task.
func (s *Service) regenerateTask(task *Task, completedAt time.Time) *Task {
	// Check if past end date
	if task.RecurEnd != nil && time.Now().After(*task.RecurEnd) {
		return nil
	}

	// Parse the recurrence rule
	rule, err := recurparse.FromJSON(*task.RecurRule)
	if err != nil {
		return nil
	}

	// Calculate next occurrence
	recurrenceType := recurparse.TypeFixed
	if *task.RecurType == RecurTypeRelative {
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
	if task.DueDate != nil {
		dueDate = &nextDate
	} else {
		plannedDate = &nextDate
	}

	// Determine the parent ID for linking
	parentID := task.RecurParentID
	if parentID == nil {
		parentID = &task.ID
	}

	// Create the next task
	opts := &CreateOptions{
		PlannedDate:   plannedDate,
		DueDate:       dueDate,
		RecurType:     task.RecurType,
		RecurRule:     task.RecurRule,
		RecurEnd:      task.RecurEnd,
		RecurParentID: parentID,
	}

	// Copy project/area by ID directly
	nextTask := &Task{
		UUID:          uuid.New().String(),
		Title:         task.Title,
		ProjectID:     task.ProjectID,
		AreaID:        task.AreaID,
		PlannedDate:   plannedDate,
		DueDate:       dueDate,
		State:         StateActive,
		Status:        StatusTodo,
		CreatedAt:     time.Now(),
		RecurType:     opts.RecurType,
		RecurRule:     opts.RecurRule,
		RecurEnd:      opts.RecurEnd,
		RecurParentID: opts.RecurParentID,
	}

	if err := s.repo.Create(nextTask); err != nil {
		return nil
	}

	// Copy tags from original task
	if len(task.Tags) > 0 {
		for _, tag := range task.Tags {
			if err := s.repo.AddTag(nextTask.ID, tag); err != nil {
				return nil
			}
		}
		nextTask.Tags = task.Tags
	}

	return nextTask
}

func (s *Service) Delete(ids []int64) ([]Task, error) {
	var deleted []Task

	for _, id := range ids {
		task, err := s.repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return deleted, ErrTaskNotFound
			}
			return deleted, err
		}
		if err := s.repo.Delete(id); err != nil {
			return deleted, err
		}
		deleted = append(deleted, *task)
	}

	return deleted, nil
}

func (s *Service) ListCompleted(since *time.Time) ([]Task, error) {
	return s.repo.ListCompleted(since)
}

func (s *Service) Defer(id int64) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.State = StateSomeday
	task.PlannedDate = nil // clear planned date when deferring

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Service) Activate(id int64) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.State = StateActive

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Service) SetPlannedDate(id int64, date *time.Time) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.PlannedDate = date

	// Setting a planned date activates a someday task
	if date != nil && task.State == StateSomeday {
		task.State = StateActive
	}

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Service) SetDueDate(id int64, date *time.Time) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.DueDate = date

	// Setting a due date activates a someday task
	if date != nil && task.State == StateSomeday {
		task.State = StateActive
	}

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SetProject assigns or clears the project for a task.
func (s *Service) SetProject(id int64, projectName string) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if projectName == "" {
		task.ProjectID = nil
	} else {
		p, err := s.projectService.GetByName(projectName)
		if err != nil {
			return nil, err
		}
		task.ProjectID = &p.ID
		// Clear area when setting project (mutual exclusivity)
		task.AreaID = nil
	}

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SetArea assigns or clears the area for a task.
func (s *Service) SetArea(id int64, areaName string) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if areaName == "" {
		task.AreaID = nil
	} else {
		a, err := s.areaService.GetByName(areaName)
		if err != nil {
			return nil, err
		}
		task.AreaID = &a.ID
		// Clear project when setting area (mutual exclusivity)
		task.ProjectID = nil
	}

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SetTitle updates the title of a task.
func (s *Service) SetTitle(id int64, title string) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.Title = title

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SetRecurrence sets or clears the recurrence rule for a task.
func (s *Service) SetRecurrence(id int64, recurType, recurRule *string, recurEnd *time.Time) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.RecurType = recurType
	task.RecurRule = recurRule
	task.RecurEnd = recurEnd

	// If setting recurrence, unpause
	if recurType != nil {
		task.RecurPaused = false
	}

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// PauseRecurrence pauses recurrence without clearing the rule.
func (s *Service) PauseRecurrence(id int64) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.RecurPaused = true

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// ResumeRecurrence resumes a paused recurrence.
func (s *Service) ResumeRecurrence(id int64) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.RecurPaused = false

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SetRecurrenceEnd sets or clears the end date for recurrence.
func (s *Service) SetRecurrenceEnd(id int64, endDate *time.Time) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.RecurEnd = endDate

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetByID returns a task by its ID.
func (s *Service) GetByID(id int64) (*Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

// AddTag adds a tag to a task.
func (s *Service) AddTag(id int64, tagName string) (*Task, error) {
	// Verify task exists
	if _, err := s.repo.GetByID(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if err := s.repo.AddTag(id, tagName); err != nil {
		return nil, err
	}

	// Reload to get updated tags
	return s.repo.GetByID(id)
}

// RemoveTag removes a tag from a task.
func (s *Service) RemoveTag(id int64, tagName string) (*Task, error) {
	// Verify task exists
	if _, err := s.repo.GetByID(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if err := s.repo.RemoveTag(id, tagName); err != nil {
		return nil, err
	}

	// Reload to get updated tags
	return s.repo.GetByID(id)
}

// ListTags returns all unique tags in use.
func (s *Service) ListTags() ([]string, error) {
	return s.repo.ListTags()
}
