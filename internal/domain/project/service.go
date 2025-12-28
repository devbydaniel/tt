package project

import (
	"github.com/devbydaniel/tt/internal/domain/area"
)

type Service struct {
	repo        *Repository
	areaService *area.Service
}

func NewService(repo *Repository, areaService *area.Service) *Service {
	return &Service{
		repo:        repo,
		areaService: areaService,
	}
}

func (s *Service) Create(name string, areaName string) (*Project, error) {
	project := &Project{
		Name: name,
	}

	if areaName != "" {
		a, err := s.areaService.GetByName(areaName)
		if err != nil {
			return nil, err
		}
		project.AreaID = &a.ID
	}

	if err := s.repo.Create(project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) List() ([]Project, error) {
	return s.repo.List()
}

func (s *Service) ListWithArea() ([]ProjectWithArea, error) {
	return s.repo.ListWithArea()
}

func (s *Service) GetByName(name string) (*Project, error) {
	return s.repo.GetByName(name)
}

func (s *Service) Delete(name string) (*Project, error) {
	project, err := s.repo.GetByName(name)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Delete(project.ID); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) Rename(oldName, newName string) (*Project, error) {
	project, err := s.repo.GetByName(oldName)
	if err != nil {
		return nil, err
	}

	project.Name = newName
	if err := s.repo.Update(project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) SetArea(projectName, areaName string) (*Project, error) {
	project, err := s.repo.GetByName(projectName)
	if err != nil {
		return nil, err
	}

	a, err := s.areaService.GetByName(areaName)
	if err != nil {
		return nil, err
	}

	project.AreaID = &a.ID
	if err := s.repo.Update(project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) ClearArea(projectName string) (*Project, error) {
	project, err := s.repo.GetByName(projectName)
	if err != nil {
		return nil, err
	}

	project.AreaID = nil
	if err := s.repo.Update(project); err != nil {
		return nil, err
	}

	return project, nil
}
