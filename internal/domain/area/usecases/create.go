package usecases

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

type CreateArea struct {
	Repo          *area.Repository
	SyncPersister SyncEventPersister
	ClientID      string
	DB            *database.DB
}

func (c *CreateArea) Execute(name string) (result *area.Area, err error) {
	err = c.DB.RunInTx(func() error {
		a := &area.Area{
			Name: name,
		}

		if err := c.Repo.Create(a); err != nil {
			return err
		}

		// Emit sync event if sync is enabled
		if c.SyncPersister != nil {
			if _, err := c.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:  c.ClientID,
				EventType: syncevent.EventTypeCreated,
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
