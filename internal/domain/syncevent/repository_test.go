package syncevent_test

import (
	"testing"
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/testutil"
)

func setupRepo(t *testing.T) *syncevent.Repository {
	t.Helper()
	db := testutil.NewTestDB(t)
	return syncevent.NewRepository(db)
}

func createTestEvent(clientID, entityUUID, eventUUID string, eventType syncevent.EventType, version int64) *syncevent.SyncEvent {
	snapshot := `{"uuid":"` + entityUUID + `","title":"Test Task","taskType":"task","state":"inbox","status":"todo","createdAt":"2024-01-01T00:00:00Z"}`
	return &syncevent.SyncEvent{
		EventUUID:    eventUUID,
		EntityType:   syncevent.EntityTypeTask,
		EntityUUID:   entityUUID,
		ClientID:     clientID,
		EventType:    eventType,
		EventVersion: version,
		Timestamp:    time.Now(),
		Snapshot:     &snapshot,
	}
}

func TestEventVersioning(t *testing.T) {
	repo := setupRepo(t)
	entityUUID := "entity-123"

	// First event for an entity should get version 1
	v1, err := repo.GetNextEventVersion(syncevent.EntityTypeTask, entityUUID)
	if err != nil {
		t.Fatalf("GetNextEventVersion() error = %v", err)
	}
	if v1 != 1 {
		t.Errorf("first version = %d, want 1", v1)
	}

	// Create event with version 1
	event1 := createTestEvent("client-1", entityUUID, "event-1", syncevent.EventTypeCreated, v1)
	if err := repo.Create(event1); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Next version should be 2
	v2, err := repo.GetNextEventVersion(syncevent.EntityTypeTask, entityUUID)
	if err != nil {
		t.Fatalf("GetNextEventVersion() error = %v", err)
	}
	if v2 != 2 {
		t.Errorf("second version = %d, want 2", v2)
	}

	// Create event with version 2
	event2 := createTestEvent("client-1", entityUUID, "event-2", syncevent.EventTypeUpdated, v2)
	if err := repo.Create(event2); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Next version should be 3
	v3, err := repo.GetNextEventVersion(syncevent.EntityTypeTask, entityUUID)
	if err != nil {
		t.Fatalf("GetNextEventVersion() error = %v", err)
	}
	if v3 != 3 {
		t.Errorf("third version = %d, want 3", v3)
	}
}

func TestEventVersioningPerEntity(t *testing.T) {
	repo := setupRepo(t)

	// Different entities should have independent version counters
	entity1 := "entity-1"
	entity2 := "entity-2"

	event1 := createTestEvent("client-1", entity1, "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-1", entity1, "event-2", syncevent.EventTypeUpdated, 2)
	event3 := createTestEvent("client-1", entity2, "event-3", syncevent.EventTypeCreated, 1)

	repo.Create(event1)
	repo.Create(event2)
	repo.Create(event3)

	// Entity 1 should be at version 3
	v1, _ := repo.GetNextEventVersion(syncevent.EntityTypeTask, entity1)
	if v1 != 3 {
		t.Errorf("entity1 next version = %d, want 3", v1)
	}

	// Entity 2 should be at version 2
	v2, _ := repo.GetNextEventVersion(syncevent.EntityTypeTask, entity2)
	if v2 != 2 {
		t.Errorf("entity2 next version = %d, want 2", v2)
	}
}

func TestPushTracking(t *testing.T) {
	repo := setupRepo(t)

	// Create 3 unpushed events
	event1 := createTestEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-1", "entity-2", "event-2", syncevent.EventTypeCreated, 1)
	event3 := createTestEvent("client-1", "entity-3", "event-3", syncevent.EventTypeCreated, 1)

	repo.Create(event1)
	repo.Create(event2)
	repo.Create(event3)

	// All 3 should be unpushed
	unpushed, err := repo.GetUnpushed(10)
	if err != nil {
		t.Fatalf("GetUnpushed() error = %v", err)
	}
	if len(unpushed) != 3 {
		t.Errorf("unpushed count = %d, want 3", len(unpushed))
	}

	// Mark first 2 as pushed
	err = repo.MarkAsPushed([]string{"event-1", "event-2"}, time.Now())
	if err != nil {
		t.Fatalf("MarkAsPushed() error = %v", err)
	}

	// Only 1 should be unpushed now
	unpushed, _ = repo.GetUnpushed(10)
	if len(unpushed) != 1 {
		t.Errorf("unpushed count after marking = %d, want 1", len(unpushed))
	}
	if unpushed[0].EventUUID != "event-3" {
		t.Errorf("remaining unpushed = %s, want event-3", unpushed[0].EventUUID)
	}
}

func TestPushTrackingBatching(t *testing.T) {
	repo := setupRepo(t)

	// Create 5 events
	for i := 1; i <= 5; i++ {
		event := createTestEvent("client-1", "entity-"+string(rune('0'+i)), "event-"+string(rune('0'+i)), syncevent.EventTypeCreated, 1)
		repo.Create(event)
	}

	// Request batch of 2
	batch, _ := repo.GetUnpushed(2)
	if len(batch) != 2 {
		t.Errorf("batch size = %d, want 2", len(batch))
	}

	// Request all
	all, _ := repo.GetUnpushed(100)
	if len(all) != 5 {
		t.Errorf("total unpushed = %d, want 5", len(all))
	}
}

func TestGetByUUID(t *testing.T) {
	repo := setupRepo(t)

	event := createTestEvent("client-1", "entity-1", "event-uuid-123", syncevent.EventTypeCreated, 1)
	repo.Create(event)

	// Should find the event
	found, err := repo.GetByUUID("event-uuid-123")
	if err != nil {
		t.Fatalf("GetByUUID() error = %v", err)
	}
	if found.EventUUID != "event-uuid-123" {
		t.Errorf("found UUID = %s, want event-uuid-123", found.EventUUID)
	}
	if found.EntityUUID != "entity-1" {
		t.Errorf("found EntityUUID = %s, want entity-1", found.EntityUUID)
	}

	// Should return error for non-existent
	_, err = repo.GetByUUID("non-existent")
	if err != syncevent.ErrEventNotFound {
		t.Errorf("GetByUUID() error = %v, want ErrEventNotFound", err)
	}
}

func TestCursorBasedPagination(t *testing.T) {
	repo := setupRepo(t)

	// Create events from multiple clients/entities
	event1 := createTestEvent("client-a", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-b", "entity-2", "event-2", syncevent.EventTypeCreated, 1)
	event3 := createTestEvent("client-a", "entity-1", "event-3", syncevent.EventTypeUpdated, 2) // Update entity-1

	repo.Create(event1)
	repo.Create(event2)
	repo.Create(event3)

	// Get initial cursor
	maxID, _ := repo.GetMaxID()
	if maxID != 3 {
		t.Errorf("maxID = %d, want 3", maxID)
	}

	// Get states from cursor 0 (all entities, but only latest per entity)
	states, newCursor, err := repo.GetLatestStatesSince(0, nil)
	if err != nil {
		t.Fatalf("GetLatestStatesSince() error = %v", err)
	}
	if newCursor != 3 {
		t.Errorf("newCursor = %d, want 3", newCursor)
	}
	if len(states) != 2 {
		t.Errorf("states count = %d, want 2 (latest per entity)", len(states))
	}

	// Entity-1 should have "updated" as latest event type
	for _, state := range states {
		if state.EntityUUID == "entity-1" {
			if state.EventType != string(syncevent.EventTypeUpdated) {
				t.Errorf("entity-1 event type = %s, want updated", state.EventType)
			}
		}
	}
}

func TestCursorExcludesEntities(t *testing.T) {
	repo := setupRepo(t)

	event1 := createTestEvent("client-a", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-b", "entity-2", "event-2", syncevent.EventTypeCreated, 1)
	event3 := createTestEvent("client-c", "entity-3", "event-3", syncevent.EventTypeCreated, 1)

	repo.Create(event1)
	repo.Create(event2)
	repo.Create(event3)

	// Exclude event-2 from results (simulates client just pushed event-2)
	// Since event-2 is the latest event for entity-2, entity-2 should be excluded
	states, _, err := repo.GetLatestStatesSince(0, []string{"event-2"})
	if err != nil {
		t.Fatalf("GetLatestStatesSince() error = %v", err)
	}

	if len(states) != 2 {
		t.Errorf("states count = %d, want 2 (entity-2 excluded)", len(states))
	}

	for _, state := range states {
		if state.EntityUUID == "entity-2" {
			t.Error("entity-2 should be excluded from results")
		}
	}
}

func TestCursorExcludesEventNotEntity(t *testing.T) {
	repo := setupRepo(t)

	// Client A creates entity-1
	event1 := createTestEvent("client-a", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	// Client B also updates entity-1 (newer event)
	event2 := createTestEvent("client-b", "entity-1", "event-2", syncevent.EventTypeUpdated, 2)

	repo.Create(event1)
	repo.Create(event2)

	// Client A pushed event-1, but event-2 (from client B) is newer.
	// Entity-1 should still be returned because its latest event (event-2) is NOT excluded.
	states, _, err := repo.GetLatestStatesSince(0, []string{"event-1"})
	if err != nil {
		t.Fatalf("GetLatestStatesSince() error = %v", err)
	}

	if len(states) != 1 {
		t.Fatalf("states count = %d, want 1 (entity-1 should be returned with third-party update)", len(states))
	}

	if states[0].EntityUUID != "entity-1" {
		t.Errorf("expected entity-1, got %s", states[0].EntityUUID)
	}
	if states[0].EventType != string(syncevent.EventTypeUpdated) {
		t.Errorf("expected updated event type, got %s", states[0].EventType)
	}
}

func TestCursorReturnsOnlyNewerEvents(t *testing.T) {
	repo := setupRepo(t)

	event1 := createTestEvent("client-a", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	repo.Create(event1)

	// Get cursor after first event
	cursor, _ := repo.GetMaxID()

	// Create more events
	event2 := createTestEvent("client-b", "entity-2", "event-2", syncevent.EventTypeCreated, 1)
	event3 := createTestEvent("client-c", "entity-3", "event-3", syncevent.EventTypeCreated, 1)
	repo.Create(event2)
	repo.Create(event3)

	// Get states since cursor - should only return entity-2 and entity-3
	states, newCursor, _ := repo.GetLatestStatesSince(cursor, nil)

	if len(states) != 2 {
		t.Errorf("states count = %d, want 2", len(states))
	}
	if newCursor != 3 {
		t.Errorf("newCursor = %d, want 3", newCursor)
	}

	for _, state := range states {
		if state.EntityUUID == "entity-1" {
			t.Error("entity-1 should not be in results (before cursor)")
		}
	}
}

func TestSyncState(t *testing.T) {
	repo := setupRepo(t)

	// Non-existent key returns empty string
	value, err := repo.GetSyncState("nonexistent")
	if err != nil {
		t.Fatalf("GetSyncState() error = %v", err)
	}
	if value != "" {
		t.Errorf("nonexistent key value = %q, want empty", value)
	}

	// Set and get
	if err := repo.SetSyncState("server_cursor", "42"); err != nil {
		t.Fatalf("SetSyncState() error = %v", err)
	}

	value, _ = repo.GetSyncState("server_cursor")
	if value != "42" {
		t.Errorf("server_cursor = %s, want 42", value)
	}

	// Update existing key
	if err := repo.SetSyncState("server_cursor", "100"); err != nil {
		t.Fatalf("SetSyncState() error = %v", err)
	}

	value, _ = repo.GetSyncState("server_cursor")
	if value != "100" {
		t.Errorf("server_cursor after update = %s, want 100", value)
	}
}

func TestDeleteAll(t *testing.T) {
	repo := setupRepo(t)

	// Create some events
	event1 := createTestEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-1", "entity-2", "event-2", syncevent.EventTypeCreated, 1)
	repo.Create(event1)
	repo.Create(event2)

	// Verify they exist
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 2 {
		t.Fatalf("should have 2 events before delete")
	}

	// Delete all
	count, err := repo.DeleteAll()
	if err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}
	if count != 2 {
		t.Errorf("deleted count = %d, want 2", count)
	}

	// Verify they're gone
	unpushed, _ = repo.GetUnpushed(10)
	if len(unpushed) != 0 {
		t.Errorf("should have 0 events after delete, got %d", len(unpushed))
	}
}

func TestDeletedEventLatestState(t *testing.T) {
	repo := setupRepo(t)

	// Create -> Update -> Delete sequence
	event1 := createTestEvent("client-a", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-a", "entity-1", "event-2", syncevent.EventTypeUpdated, 2)

	// Delete event has no snapshot
	deleteEvent := &syncevent.SyncEvent{
		EventUUID:    "event-3",
		EntityType:   syncevent.EntityTypeTask,
		EntityUUID:   "entity-1",
		ClientID:     "client-a",
		EventType:    syncevent.EventTypeDeleted,
		EventVersion: 3,
		Timestamp:    time.Now(),
		Snapshot:     nil, // Deletes have no snapshot
	}

	repo.Create(event1)
	repo.Create(event2)
	repo.Create(deleteEvent)

	// Latest state for entity-1 should be "deleted" with nil snapshot
	states, _, _ := repo.GetLatestStatesSince(0, nil)

	if len(states) != 1 {
		t.Fatalf("states count = %d, want 1", len(states))
	}

	state := states[0]
	if state.EntityUUID != "entity-1" {
		t.Errorf("entity UUID = %s, want entity-1", state.EntityUUID)
	}
	if state.EventType != string(syncevent.EventTypeDeleted) {
		t.Errorf("event type = %s, want deleted", state.EventType)
	}
	if state.Snapshot != nil {
		t.Error("deleted entity should have nil snapshot")
	}
}

func TestIncrementFailureCount(t *testing.T) {
	repo := setupRepo(t)

	event := createTestEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	repo.Create(event)

	// Increment once
	err := repo.IncrementFailureCount([]string{"event-1"})
	if err != nil {
		t.Fatalf("IncrementFailureCount() error = %v", err)
	}

	// Should still be in unpushed (failure_count=1, threshold=3)
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 1 {
		t.Errorf("should still have 1 unpushed after 1 failure, got %d", len(unpushed))
	}
}

func TestPermanentlyFailedExcludedFromUnpushed(t *testing.T) {
	repo := setupRepo(t)

	event := createTestEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	repo.Create(event)

	// Increment failure count to threshold
	for i := 0; i < syncevent.MaxFailureCount; i++ {
		if err := repo.IncrementFailureCount([]string{"event-1"}); err != nil {
			t.Fatalf("IncrementFailureCount() error = %v", err)
		}
	}

	// Should be excluded from unpushed
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 0 {
		t.Errorf("permanently failed event should be excluded from unpushed, got %d", len(unpushed))
	}

	// Should appear in permanently failed list
	failed, err := repo.GetPermanentlyFailed()
	if err != nil {
		t.Fatalf("GetPermanentlyFailed() error = %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("should have 1 permanently failed event, got %d", len(failed))
	}
}

func TestDeletePermanentlyFailed(t *testing.T) {
	repo := setupRepo(t)

	event1 := createTestEvent("client-1", "entity-1", "event-1", syncevent.EventTypeCreated, 1)
	event2 := createTestEvent("client-1", "entity-2", "event-2", syncevent.EventTypeCreated, 1)
	repo.Create(event1)
	repo.Create(event2)

	// Mark event-1 as permanently failed
	for i := 0; i < syncevent.MaxFailureCount; i++ {
		repo.IncrementFailureCount([]string{"event-1"})
	}

	// Delete permanently failed
	count, err := repo.DeletePermanentlyFailed()
	if err != nil {
		t.Fatalf("DeletePermanentlyFailed() error = %v", err)
	}
	if count != 1 {
		t.Errorf("deleted count = %d, want 1", count)
	}

	// event-2 should still be unpushed
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 1 {
		t.Errorf("should have 1 unpushed after cleaning failed, got %d", len(unpushed))
	}
	if unpushed[0].EventUUID != "event-2" {
		t.Errorf("remaining event = %s, want event-2", unpushed[0].EventUUID)
	}
}

func TestIncrementFailureCountEmptySlice(t *testing.T) {
	repo := setupRepo(t)

	err := repo.IncrementFailureCount([]string{})
	if err != nil {
		t.Errorf("IncrementFailureCount(empty) should not error, got %v", err)
	}
}
