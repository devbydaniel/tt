package usecases

import (
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
)

const (
	// DefaultBatchSize is the default number of events to push in a single batch.
	DefaultBatchSize = 100
)

// PushEvents orchestrates pushing local sync events to the remote server.
type PushEvents struct {
	Repo     *syncevent.Repository
	Client   *syncevent.Client
	ClientID string
}

// PushResult contains the result of pushing events to the server.
type PushResult struct {
	Pushed   int
	Rejected int
	Errors   []string
}

// Execute pushes unpushed events to the sync server.
func (p *PushEvents) Execute() (*PushResult, error) {
	result := &PushResult{}

	for {
		// Get batch of unpushed events
		events, err := p.Repo.GetUnpushed(DefaultBatchSize)
		if err != nil {
			return result, err
		}

		// No more events to push
		if len(events) == 0 {
			break
		}

		// Push to server
		resp, err := p.Client.PushEvents(p.ClientID, events)
		if err != nil {
			return result, err
		}

		// Mark accepted events as pushed
		if len(resp.Accepted) > 0 {
			if err := p.Repo.MarkAsPushed(resp.Accepted, time.Now()); err != nil {
				return result, err
			}
			result.Pushed += len(resp.Accepted)
		}

		// Record rejected events
		for _, rejected := range resp.Rejected {
			result.Rejected++
			result.Errors = append(result.Errors, rejected.EventUUID+": "+rejected.Reason)
		}

		// Stop if no progress was made (all events rejected — avoids infinite loop)
		if len(resp.Accepted) == 0 {
			break
		}

		// If we got less than batch size, we're done
		if len(events) < DefaultBatchSize {
			break
		}
	}

	return result, nil
}
