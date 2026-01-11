package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

type CreateArea struct {
	Repo          *area.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

func (c *CreateArea) Execute(name string) (*area.Area, error) {
	a := &area.Area{
		Name: name,
	}

	if err := c.Repo.Create(a); err != nil {
		return nil, err
	}

	// Emit sync event if sync is enabled
	if c.SyncPersister != nil {
		c.SyncPersister.Execute(&synceventusecases.PersistOptions{
			ClientID:  c.ClientID,
			EventType: syncevent.EventTypeCreated,
			Area:      a,
		})
	}

	return a, nil
}
