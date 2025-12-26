package task

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/t/internal/domain/area"
	"github.com/devbydaniel/t/internal/domain/project"
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
	Someday     bool // if true, create in someday state
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

	return task, nil
}

type ListOptions struct {
	ProjectName string
	AreaName    string
	Today       bool   // show today + overdue
	Upcoming    bool   // show future dates
	Someday     bool   // show someday tasks
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

		if opts.Someday {
			filter.State = StateSomeday
		} else if opts.Today {
			filter.Today = true
		} else if opts.Upcoming {
			filter.Upcoming = true
		}
		// opts.All means no date filter, just state = active (default)
	}

	return s.repo.List(filter)
}

func (s *Service) Complete(ids []int64) ([]Task, error) {
	completedAt := time.Now()
	var completed []Task

	for _, id := range ids {
		if err := s.repo.Complete(id, completedAt); err != nil {
			return completed, err
		}
		task, err := s.repo.GetByID(id)
		if err != nil {
			return completed, err
		}
		completed = append(completed, *task)
	}

	return completed, nil
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
