package usecases

import (
	"strconv"
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
)

const (
	// SyncStateServerCursor is the key for storing the server cursor in sync_state.
	SyncStateServerCursor = "server_cursor"
)

// SyncEvents orchestrates bidirectional sync with the server.
type SyncEvents struct {
	Repo     *syncevent.Repository
	Client   *syncevent.Client
	ClientID string
	Applier  EntityApplier
}

// EntityApplier applies entity states received from the server.
type EntityApplier interface {
	Apply(entities []syncevent.EntityState) (*ApplyResult, error)
}

// SyncEventsResult contains the result of syncing events.
type SyncEventsResult struct {
	Pushed   int
	Pulled   int
	Applied  int
	Rejected int
	Errors   []string
}

// Execute performs bidirectional sync with the server.
func (s *SyncEvents) Execute() (*SyncEventsResult, error) {
	result := &SyncEventsResult{
		Errors: make([]string, 0),
	}

	// Get current cursor from sync_state (default 0)
	cursor, err := s.getCursor()
	if err != nil {
		return nil, err
	}

	for {
		// Get batch of unpushed local events
		events, err := s.Repo.GetUnpushed(DefaultBatchSize)
		if err != nil {
			return result, err
		}

		// Call server sync endpoint
		resp, err := s.Client.Sync(s.ClientID, cursor, events)
		if err != nil {
			return result, err
		}

		// Mark accepted events as pushed
		if len(resp.Accepted) > 0 {
			if err := s.Repo.MarkAsPushed(resp.Accepted, time.Now()); err != nil {
				return result, err
			}
			result.Pushed += len(resp.Accepted)
		}

		// Record rejected events and track failure counts
		if len(resp.Rejected) > 0 {
			rejectedUUIDs := make([]string, len(resp.Rejected))
			for i, rejected := range resp.Rejected {
				result.Rejected++
				result.Errors = append(result.Errors, rejected.EventUUID+": "+rejected.Reason)
				rejectedUUIDs[i] = rejected.EventUUID
			}
			if err := s.Repo.IncrementFailureCount(rejectedUUIDs); err != nil {
				return result, err
			}
		}

		// Apply received entity states
		if len(resp.Entities) > 0 {
			result.Pulled += len(resp.Entities)
			applyResult, err := s.Applier.Apply(resp.Entities)
			if err != nil {
				return result, err
			}
			result.Applied += applyResult.Applied
		}

		// Update cursor
		if resp.NewCursor > cursor {
			if err := s.setCursor(resp.NewCursor); err != nil {
				return result, err
			}
			cursor = resp.NewCursor
		}

		// Stop if no events were sent (nothing left to push)
		// or if no progress was made (all events rejected — avoids infinite loop)
		if len(events) == 0 || len(resp.Accepted) == 0 {
			break
		}

		// If we pushed less than batch size, we're done
		if len(events) < DefaultBatchSize {
			break
		}
	}

	return result, nil
}

// getCursor retrieves the current sync cursor from sync_state.
func (s *SyncEvents) getCursor() (int64, error) {
	value, err := s.Repo.GetSyncState(SyncStateServerCursor)
	if err != nil {
		return 0, err
	}
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

// setCursor stores the sync cursor in sync_state.
func (s *SyncEvents) setCursor(cursor int64) error {
	return s.Repo.SetSyncState(SyncStateServerCursor, strconv.FormatInt(cursor, 10))
}
