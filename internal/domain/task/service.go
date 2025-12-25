package task

import (
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(title string) (*Task, error) {
	task := &Task{
		UUID:      uuid.New().String(),
		Title:     title,
		Status:    StatusTodo,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Service) List() ([]Task, error) {
	return s.repo.List()
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

func (s *Service) ListCompleted(since *time.Time) ([]Task, error) {
	return s.repo.ListCompleted(since)
}
