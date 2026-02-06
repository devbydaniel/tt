package usecases

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

type RenameArea struct {
	Repo          *area.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (r *RenameArea) Execute(oldName, newName string) (result *area.Area, err error) {
	err = r.DB.RunInTx(func() error {
		a, err := r.Repo.GetByName(oldName)
		if err != nil {
			return err
		}

		a.Name = newName
		if err := r.Repo.Update(a); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if r.SyncPersister != nil {
			if _, err := r.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  r.ClientID,
				EventType: syncevent.EventTypeUpdated,
				Area:      a,
			}); err != nil {
				return err
			}
		}

		result = a
		return nil
	})
	return
}
