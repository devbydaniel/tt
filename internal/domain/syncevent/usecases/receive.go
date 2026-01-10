package usecases

import (
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
)

// ReceiveEvents handles receiving and persisting events from clients.
type ReceiveEvents struct {
	Repo *syncevent.Repository
}

// ReceiveRequest contains the events to receive from a client.
type ReceiveRequest struct {
	ClientID string
	Events   []*syncevent.SyncEvent
}

// RejectedEvent represents an event that was rejected.
type RejectedEvent struct {
	EventUUID string `json:"eventUuid"`
	Reason    string `json:"reason"`
}

// ReceiveResult contains the result of receiving events.
type ReceiveResult struct {
	Accepted []string
	Rejected []RejectedEvent
}

// Execute receives and persists events from a client.
func (r *ReceiveEvents) Execute(req *ReceiveRequest) (*ReceiveResult, error) {
	result := &ReceiveResult{
		Accepted: make([]string, 0),
		Rejected: make([]RejectedEvent, 0),
	}

	for _, event := range req.Events {
		// Validate client ID matches
		if event.ClientID != req.ClientID {
			result.Rejected = append(result.Rejected, RejectedEvent{
				EventUUID: event.EventUUID,
				Reason:    "client_id mismatch",
			})
			continue
		}

		// Check for duplicate
		existing, err := r.Repo.GetByUUID(event.EventUUID)
		if err != nil && !errors.Is(err, syncevent.ErrEventNotFound) {
			return nil, err
		}
		if existing != nil {
			result.Rejected = append(result.Rejected, RejectedEvent{
				EventUUID: event.EventUUID,
				Reason:    "duplicate",
			})
			continue
		}

		// Persist the event
		if err := r.Repo.Create(event); err != nil {
			return nil, err
		}

		result.Accepted = append(result.Accepted, event.EventUUID)
	}

	return result, nil
}
