package usecases

import (
	"errors"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
)

// ReceiveEvents handles receiving and persisting events from clients.
type ReceiveEvents struct {
	Repo    *syncevent.Repository
	Applier EntityApplier // Optional: applies events to tasks/areas tables
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

	// Collect entity states for batch apply after the loop
	var pendingStates []syncevent.EntityState

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

		// Collect entity state for batch apply
		if r.Applier != nil {
			pendingStates = append(pendingStates, syncevent.EntityState{
				EntityType: string(event.EntityType),
				EntityUUID: event.EntityUUID,
				EventType:  string(event.EventType),
				Snapshot:   event.Snapshot,
			})
		}

		result.Accepted = append(result.Accepted, event.EventUUID)
	}

	// Batch-apply all accepted events so sorting logic in Apply works correctly
	if r.Applier != nil && len(pendingStates) > 0 {
		// Ignore apply errors - events are already stored
		_, _ = r.Applier.Apply(pendingStates)
	}

	return result, nil
}

// SyncRequest contains the request for bidirectional sync.
type SyncRequest struct {
	ClientID string
	Cursor   int64
	Events   []*syncevent.SyncEvent
}

// SyncResult contains the result of bidirectional sync.
type SyncResult struct {
	Accepted  []string
	Rejected  []RejectedEvent
	Entities  []syncevent.EntityState
	NewCursor int64
}

// ExecuteSync handles bidirectional sync: receives events and returns latest states.
func (r *ReceiveEvents) ExecuteSync(req *SyncRequest) (*SyncResult, error) {
	result := &SyncResult{
		Accepted: make([]string, 0),
		Rejected: make([]RejectedEvent, 0),
	}

	// Track which entity UUIDs we just received (to exclude from response)
	acceptedEntityUUIDs := make([]string, 0)

	// Collect entity states for batch apply after the loop
	var pendingStates []syncevent.EntityState

	// Process incoming events
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

		// Collect entity state for batch apply
		if r.Applier != nil {
			pendingStates = append(pendingStates, syncevent.EntityState{
				EntityType: string(event.EntityType),
				EntityUUID: event.EntityUUID,
				EventType:  string(event.EventType),
				Snapshot:   event.Snapshot,
			})
		}

		result.Accepted = append(result.Accepted, event.EventUUID)
		acceptedEntityUUIDs = append(acceptedEntityUUIDs, event.EntityUUID)
	}

	// Batch-apply all accepted events so sorting logic in Apply works correctly
	if r.Applier != nil && len(pendingStates) > 0 {
		// Ignore apply errors - events are already stored
		_, _ = r.Applier.Apply(pendingStates)
	}

	// Get latest states since cursor, excluding entities we just received
	entities, newCursor, err := r.Repo.GetLatestStatesSince(req.Cursor, acceptedEntityUUIDs)
	if err != nil {
		return nil, err
	}

	result.Entities = entities
	result.NewCursor = newCursor

	return result, nil
}
