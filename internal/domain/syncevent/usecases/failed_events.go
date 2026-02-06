package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/syncevent"
)

// ListFailedEvents returns all permanently failed sync events.
type ListFailedEvents struct {
	Repo *syncevent.Repository
}

func (l *ListFailedEvents) Execute() ([]*syncevent.SyncEvent, error) {
	return l.Repo.GetPermanentlyFailed()
}

// CleanFailedEvents removes all permanently failed sync events.
type CleanFailedEvents struct {
	Repo *syncevent.Repository
}

// CleanFailedResult contains the result of cleaning failed events.
type CleanFailedResult struct {
	Deleted int64
}

func (c *CleanFailedEvents) Execute() (*CleanFailedResult, error) {
	count, err := c.Repo.DeletePermanentlyFailed()
	if err != nil {
		return nil, err
	}
	return &CleanFailedResult{Deleted: count}, nil
}
