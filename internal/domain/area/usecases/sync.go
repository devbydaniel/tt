package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

// SyncEventPersister creates sync events for area changes.
// When nil, sync events are not emitted (sync is disabled).
type SyncEventPersister interface {
	Execute(opts *synceventusecases.PersistOptions) (*syncevent.SyncEvent, error)
}
