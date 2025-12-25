package area

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(name string) (*Area, error) {
	area := &Area{
		Name: name,
	}

	if err := s.repo.Create(area); err != nil {
		return nil, err
	}

	return area, nil
}

func (s *Service) List() ([]Area, error) {
	return s.repo.List()
}

func (s *Service) GetByName(name string) (*Area, error) {
	return s.repo.GetByName(name)
}

func (s *Service) Delete(name string) (*Area, error) {
	area, err := s.repo.GetByName(name)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Delete(area.ID); err != nil {
		return nil, err
	}

	return area, nil
}
