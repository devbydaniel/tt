package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

type DeleteArea struct {
	Repo          *area.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (d *DeleteArea) Execute(name string) (*area.Area, error) {
	a, err := d.Repo.GetByName(name)
	if err != nil {
		return nil, err
	}

	if err := d.Repo.Delete(a.ID); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if d.SyncPersister != nil {
		d.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:   d.ClientID,
			EventType:  syncevent.EventTypeDeleted,
			Area:       a,
			EntityUUID: a.UUID,
		})
	}

	return a, nil
}
