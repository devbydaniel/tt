package usecases_test

import (
	"testing"
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/testutil"
)

func setupReceiveEvents(t *testing.T) (*usecases.ReceiveEvents, *syncevent.Repository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := syncevent.NewRepository(db)
	return &usecases.ReceiveEvents{Repo: repo}, repo
}

func createEvent(clientID, entityUUID, eventUUID string, eventType syncevent.EventType) *syncevent.SyncEvent {
	snapshot := `{"uuid":"` + entityUUID + `","title":"Test","taskType":"task","state":"inbox","status":"todo","createdAt":"2024-01-01T00:00:00Z"}`
	return &syncevent.SyncEvent{
		EventUUID:    eventUUID,
		EntityType:   syncevent.EntityTypeTask,
		EntityUUID:   entityUUID,
		ClientID:     clientID,
		EventType:    eventType,
		EventVersion: 1,
		Timestamp:    time.Now(),
		Snapshot:     &snapshot,
	}
}

func TestReceiveEventsAcceptsValidEvents(t *testing.T) {
	receive, repo := setupReceiveEvents(t)

	req := &usecases.ReceiveRequest{
		ClientID: "client-1",
		Events: []*syncevent.SyncEvent{
			createEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated),
			createEvent("client-1", "entity-2", "event-2", syncevent.EventTypeCreated),
		},
	}

	result, err := receive.Execute(req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Accepted) != 2 {
		t.Errorf("accepted count = %d, want 2", len(result.Accepted))
	}
	if len(result.Rejected) != 0 {
		t.Errorf("rejected count = %d, want 0", len(result.Rejected))
	}

	// Verify events were persisted
	event1, _ := repo.GetByUUID("event-1")
	if event1 == nil {
		t.Error("event-1 should be persisted")
	}
	event2, _ := repo.GetByUUID("event-2")
	if event2 == nil {
		t.Error("event-2 should be persisted")
	}
}

func TestReceiveEventsRejectsClientIDMismatch(t *testing.T) {
	receive, _ := setupReceiveEvents(t)

	req := &usecases.ReceiveRequest{
		ClientID: "client-1",
		Events: []*syncevent.SyncEvent{
			createEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated),
			createEvent("client-2", "entity-2", "event-2", syncevent.EventTypeCreated), // Wrong client
		},
	}

	result, err := receive.Execute(req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Accepted) != 1 {
		t.Errorf("accepted count = %d, want 1", len(result.Accepted))
	}
	if len(result.Rejected) != 1 {
		t.Errorf("rejected count = %d, want 1", len(result.Rejected))
	}

	if result.Rejected[0].EventUUID != "event-2" {
		t.Errorf("rejected event = %s, want event-2", result.Rejected[0].EventUUID)
	}
	if result.Rejected[0].Reason != "client_id mismatch" {
		t.Errorf("rejection reason = %s, want 'client_id mismatch'", result.Rejected[0].Reason)
	}
}

func TestReceiveEventsRejectsDuplicates(t *testing.T) {
	receive, _ := setupReceiveEvents(t)

	// First request - should accept
	req1 := &usecases.ReceiveRequest{
		ClientID: "client-1",
		Events: []*syncevent.SyncEvent{
			createEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated),
		},
	}
	result1, _ := receive.Execute(req1)
	if len(result1.Accepted) != 1 {
		t.Fatalf("first request should accept, got %d accepted", len(result1.Accepted))
	}

	// Second request with same event UUID - should reject as duplicate
	req2 := &usecases.ReceiveRequest{
		ClientID: "client-1",
		Events: []*syncevent.SyncEvent{
			createEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated), // Same UUID
		},
	}
	result2, err := receive.Execute(req2)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result2.Accepted) != 0 {
		t.Errorf("accepted count = %d, want 0", len(result2.Accepted))
	}
	if len(result2.Rejected) != 1 {
		t.Errorf("rejected count = %d, want 1", len(result2.Rejected))
	}
	if result2.Rejected[0].Reason != "duplicate" {
		t.Errorf("rejection reason = %s, want 'duplicate'", result2.Rejected[0].Reason)
	}
}

func TestReceiveEventsIdempotentRetry(t *testing.T) {
	receive, repo := setupReceiveEvents(t)

	// Simulate a client retrying a batch where some events already succeeded
	event1 := createEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated)
	event2 := createEvent("client-1", "entity-2", "event-2", syncevent.EventTypeCreated)

	// First request succeeds for event-1 only
	req1 := &usecases.ReceiveRequest{
		ClientID: "client-1",
		Events:   []*syncevent.SyncEvent{event1},
	}
	receive.Execute(req1)

	// Client retries with both events
	req2 := &usecases.ReceiveRequest{
		ClientID: "client-1",
		Events:   []*syncevent.SyncEvent{event1, event2},
	}
	result2, _ := receive.Execute(req2)

	// event-1 rejected as duplicate, event-2 accepted
	if len(result2.Accepted) != 1 {
		t.Errorf("accepted count = %d, want 1", len(result2.Accepted))
	}
	if len(result2.Rejected) != 1 {
		t.Errorf("rejected count = %d, want 1", len(result2.Rejected))
	}

	// Both events should exist in database
	e1, _ := repo.GetByUUID("event-1")
	e2, _ := repo.GetByUUID("event-2")
	if e1 == nil || e2 == nil {
		t.Error("both events should exist in database after retry")
	}
}

func TestExecuteSyncReturnsLatestStates(t *testing.T) {
	receive, repo := setupReceiveEvents(t)

	// Pre-populate with events from another client
	existingEvent := createEvent("client-other", "entity-existing", "event-existing", syncevent.EventTypeCreated)
	repo.Create(existingEvent)

	// Client syncs with their events
	req := &usecases.SyncRequest{
		ClientID: "client-1",
		Cursor:   0,
		Events: []*syncevent.SyncEvent{
			createEvent("client-1", "entity-new", "event-new", syncevent.EventTypeCreated),
		},
	}

	result, err := receive.ExecuteSync(req)
	if err != nil {
		t.Fatalf("ExecuteSync() error = %v", err)
	}

	// New event should be accepted
	if len(result.Accepted) != 1 {
		t.Errorf("accepted count = %d, want 1", len(result.Accepted))
	}

	// Should return the existing entity state (not the one client just pushed)
	if len(result.Entities) != 1 {
		t.Errorf("entities count = %d, want 1", len(result.Entities))
	}

	// The returned entity should be the pre-existing one, not the one client pushed
	if result.Entities[0].EntityUUID != "entity-existing" {
		t.Errorf("entity UUID = %s, want entity-existing", result.Entities[0].EntityUUID)
	}
}

func TestExecuteSyncExcludesJustPushedEntities(t *testing.T) {
	receive, repo := setupReceiveEvents(t)

	// Pre-populate with two entities from other clients
	repo.Create(createEvent("client-other", "entity-1", "event-1", syncevent.EventTypeCreated))
	repo.Create(createEvent("client-other", "entity-2", "event-2", syncevent.EventTypeCreated))

	// Client pushes an update to entity-1
	req := &usecases.SyncRequest{
		ClientID: "client-1",
		Cursor:   0,
		Events: []*syncevent.SyncEvent{
			createEvent("client-1", "entity-1", "event-update", syncevent.EventTypeUpdated),
		},
	}

	result, _ := receive.ExecuteSync(req)

	// Should only return entity-2, not entity-1 (which client just pushed)
	if len(result.Entities) != 1 {
		t.Fatalf("entities count = %d, want 1", len(result.Entities))
	}
	if result.Entities[0].EntityUUID != "entity-2" {
		t.Errorf("entity UUID = %s, want entity-2", result.Entities[0].EntityUUID)
	}
}

func TestExecuteSyncCursorPagination(t *testing.T) {
	receive, repo := setupReceiveEvents(t)

	// Create initial events
	repo.Create(createEvent("client-other", "entity-1", "event-1", syncevent.EventTypeCreated))
	repo.Create(createEvent("client-other", "entity-2", "event-2", syncevent.EventTypeCreated))

	// First sync from cursor 0
	req1 := &usecases.SyncRequest{
		ClientID: "client-1",
		Cursor:   0,
		Events:   []*syncevent.SyncEvent{},
	}
	result1, _ := receive.ExecuteSync(req1)

	if len(result1.Entities) != 2 {
		t.Errorf("first sync entities = %d, want 2", len(result1.Entities))
	}
	cursor := result1.NewCursor

	// Add more events
	repo.Create(createEvent("client-other", "entity-3", "event-3", syncevent.EventTypeCreated))

	// Second sync from previous cursor
	req2 := &usecases.SyncRequest{
		ClientID: "client-1",
		Cursor:   cursor,
		Events:   []*syncevent.SyncEvent{},
	}
	result2, _ := receive.ExecuteSync(req2)

	// Should only return entity-3
	if len(result2.Entities) != 1 {
		t.Errorf("second sync entities = %d, want 1", len(result2.Entities))
	}
	if result2.Entities[0].EntityUUID != "entity-3" {
		t.Errorf("entity UUID = %s, want entity-3", result2.Entities[0].EntityUUID)
	}
}

func TestExecuteSyncReturnsLatestEventPerEntity(t *testing.T) {
	receive, repo := setupReceiveEvents(t)

	// Create sequence of events for same entity
	repo.Create(createEvent("client-other", "entity-1", "event-1", syncevent.EventTypeCreated))
	repo.Create(createEvent("client-other", "entity-1", "event-2", syncevent.EventTypeUpdated))
	repo.Create(createEvent("client-other", "entity-1", "event-3", syncevent.EventTypeCompleted))

	req := &usecases.SyncRequest{
		ClientID: "client-1",
		Cursor:   0,
		Events:   []*syncevent.SyncEvent{},
	}
	result, _ := receive.ExecuteSync(req)

	// Should only return 1 entity state (the latest)
	if len(result.Entities) != 1 {
		t.Fatalf("entities count = %d, want 1", len(result.Entities))
	}

	if result.Entities[0].EventType != string(syncevent.EventTypeCompleted) {
		t.Errorf("event type = %s, want completed", result.Entities[0].EventType)
	}
}
