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
