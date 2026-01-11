package usecases

import "github.com/devbydaniel/tt/internal/domain/syncevent"

// ResetSync clears all local sync events and state.
type ResetSync struct {
	Repo *syncevent.Repository
}

// Execute deletes all sync events and resets the cursor, returns the event count.
func (r *ResetSync) Execute() (int64, error) {
	count, err := r.Repo.DeleteAll()
	if err != nil {
		return count, err
	}

	// Also reset the sync cursor so next sync starts fresh
	if err := r.Repo.SetSyncState(SyncStateServerCursor, "0"); err != nil {
		return count, err
	}

	return count, nil
}
