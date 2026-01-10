package usecases

import "github.com/devbydaniel/tt/internal/domain/syncevent"

// ResetSync clears all local sync events.
type ResetSync struct {
	Repo *syncevent.Repository
}

// Execute deletes all sync events and returns the count.
func (r *ResetSync) Execute() (int64, error) {
	return r.Repo.DeleteAll()
}
