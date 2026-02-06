package usecases

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// ProjectLookupForSetProject is what this use case needs to look up projects (which are now tasks)
type ProjectLookupForSetProject interface {
	Execute(name string) (*task.Task, error)
}

type SetTaskProject struct {
	Repo          *task.Repository
	ProjectLookup ProjectLookupForSetProject
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (s *SetTaskProject) Execute(id int64, projectName string) (result *task.Task, err error) {
	err = s.DB.RunInTx(func() error {
		t, err := s.Repo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return task.ErrTaskNotFound
			}
			return err
		}

		if projectName == "" {
			t.ParentID = nil
		} else {
			p, err := s.ProjectLookup.Execute(projectName)
			if err != nil {
				return err
			}
			t.ParentID = &p.ID
			// Clear area when setting project (mutual exclusivity)
			t.AreaID = nil
		}

		if err := s.Repo.Update(t); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if s.SyncPersister != nil {
			if _, err := s.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  s.ClientID,
				EventType: syncevent.EventTypeUpdated,
				Task:      t,
			}); err != nil {
				return err
			}
		}

		result = t
		return nil
	})
	return
}
