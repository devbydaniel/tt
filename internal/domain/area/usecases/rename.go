package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

type RenameArea struct {
	Repo          *area.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (r *RenameArea) Execute(oldName, newName string) (*area.Area, error) {
	a, err := r.Repo.GetByName(oldName)
	if err != nil {
		return nil, err
	}

	a.Name = newName
	if err := r.Repo.Update(a); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if r.SyncPersister != nil {
		_, _ = r.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  r.ClientID,
			EventType: syncevent.EventTypeUpdated,
			Area:      a,
		})
	}

	return a, nil
}
