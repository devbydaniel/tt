package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/domain/task"
)

// ProjectLookupForSetProject is what this use case needs to look up projects (which are now tasks)
type ProjectLookupForSetProject interface {
	Execute(name string) (*task.Task, error)
}

type SetTaskProject struct {
	Repo          *task.Repository
	ProjectLookup ProjectLookupForSetProject
}

func (s *SetTaskProject) Execute(id int64, projectName string) (*task.Task, error) {
	t, err := s.Repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, task.ErrTaskNotFound
		}
		return nil, err
	}

	if projectName == "" {
		t.ParentID = nil
	} else {
		p, err := s.ProjectLookup.Execute(projectName)
		if err != nil {
			return nil, err
		}
		t.ParentID = &p.ID
		// Clear area when setting project (mutual exclusivity)
		t.AreaID = nil
	}

	if err := s.Repo.Update(t); err != nil {
		return nil, err
	}

	return t, nil
}
