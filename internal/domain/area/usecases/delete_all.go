package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

// DeleteAllAreas removes all areas from the database.
type DeleteAllAreas struct {
	Repo          *area.Repository
	SyncPersister SyncEventPersister
	ClientID      string
}

// Execute deletes all areas. If skipSyncEvents is true, no sync events are
// emitted (used during server reset where all sync state is being cleared).
func (d *DeleteAllAreas) Execute(skipSyncEvents ...bool) (int64, error) {
	skip := len(skipSyncEvents) > 0 && skipSyncEvents[0]

	// Emit sync events for each area before deleting
	if !skip && d.SyncPersister != nil {
		areas, err := d.Repo.List()
		if err != nil {
			return 0, err
		}
		for i := range areas {
			_, _ = d.SyncPersister.Execute(&synceventusecases.PersistOptions{
				ClientID:   d.ClientID,
				EventType:  syncevent.EventTypeDeleted,
				Area:       &areas[i],
				EntityUUID: areas[i].UUID,
			})
		}
	}

	return d.Repo.DeleteAll()
}
