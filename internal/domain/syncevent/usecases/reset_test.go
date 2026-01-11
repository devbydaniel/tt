package usecases_test

import (
	"testing"
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/testutil"
)

func setupResetSync(t *testing.T) (*usecases.ResetSync, *syncevent.Repository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := syncevent.NewRepository(db)
	return &usecases.ResetSync{Repo: repo}, repo
}

func TestResetDeletesAllEvents(t *testing.T) {
	reset, repo := setupResetSync(t)

	// Create some events
	for i := 0; i < 5; i++ {
		snapshot := `{"uuid":"entity","title":"Test","taskType":"task","state":"inbox","status":"todo","createdAt":"2024-01-01T00:00:00Z"}`
		event := &syncevent.SyncEvent{
			EventUUID:    "event-" + string(rune('0'+i)),
			EntityType:   syncevent.EntityTypeTask,
			EntityUUID:   "entity-" + string(rune('0'+i)),
			ClientID:     "client-1",
			EventType:    syncevent.EventTypeCreated,
			EventVersion: 1,
			Timestamp:    time.Now(),
			Snapshot:     &snapshot,
		}
		repo.Create(event)
	}

	// Verify events exist
	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 5 {
		t.Fatalf("should have 5 events before reset, got %d", len(unpushed))
	}

	count, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if count != 5 {
		t.Errorf("deleted count = %d, want 5", count)
	}

	// Verify all events deleted
	unpushed, _ = repo.GetUnpushed(100)
	if len(unpushed) != 0 {
		t.Errorf("should have 0 events after reset, got %d", len(unpushed))
	}
}

func TestResetClearsCursor(t *testing.T) {
	reset, repo := setupResetSync(t)

	// Set a cursor value
	repo.SetSyncState(usecases.SyncStateServerCursor, "12345")

	// Verify cursor is set
	cursor, _ := repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "12345" {
		t.Fatalf("cursor should be 12345 before reset")
	}

	reset.Execute()

	// Verify cursor is reset to 0
	cursor, _ = repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "0" {
		t.Errorf("cursor = %s, want '0'", cursor)
	}
}

func TestResetEmptyDatabase(t *testing.T) {
	reset, _ := setupResetSync(t)

	// Should not error on empty database
	count, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if count != 0 {
		t.Errorf("deleted count = %d, want 0 (empty db)", count)
	}
}

func TestResetDeletesBothPushedAndUnpushed(t *testing.T) {
	reset, repo := setupResetSync(t)

	// Create pushed and unpushed events
	snapshot := `{"uuid":"entity","title":"Test","taskType":"task","state":"inbox","status":"todo","createdAt":"2024-01-01T00:00:00Z"}`

	unpushedEvent := &syncevent.SyncEvent{
		EventUUID:    "event-unpushed",
		EntityType:   syncevent.EntityTypeTask,
		EntityUUID:   "entity-1",
		ClientID:     "client-1",
		EventType:    syncevent.EventTypeCreated,
		EventVersion: 1,
		Timestamp:    time.Now(),
		Snapshot:     &snapshot,
	}
	repo.Create(unpushedEvent)

	pushedEvent := &syncevent.SyncEvent{
		EventUUID:    "event-pushed",
		EntityType:   syncevent.EntityTypeTask,
		EntityUUID:   "entity-2",
		ClientID:     "client-1",
		EventType:    syncevent.EventTypeCreated,
		EventVersion: 1,
		Timestamp:    time.Now(),
		Snapshot:     &snapshot,
	}
	repo.Create(pushedEvent)
	repo.MarkAsPushed([]string{"event-pushed"}, time.Now())

	// Verify state
	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 1 {
		t.Fatalf("should have 1 unpushed before reset")
	}

	count, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if count != 2 {
		t.Errorf("deleted count = %d, want 2 (1 pushed + 1 unpushed)", count)
	}

	// Both events should be gone
	_, err = repo.GetByUUID("event-unpushed")
	if err != syncevent.ErrEventNotFound {
		t.Error("unpushed event should be deleted")
	}
	_, err = repo.GetByUUID("event-pushed")
	if err != syncevent.ErrEventNotFound {
		t.Error("pushed event should be deleted")
	}
}

func TestResetReturnsCount(t *testing.T) {
	reset, repo := setupResetSync(t)

	// Create specific number of events
	for i := 0; i < 7; i++ {
		snapshot := `{}`
		event := &syncevent.SyncEvent{
			EventUUID:    "event-" + string(rune('a'+i)),
			EntityType:   syncevent.EntityTypeTask,
			EntityUUID:   "entity-" + string(rune('a'+i)),
			ClientID:     "client-1",
			EventType:    syncevent.EventTypeCreated,
			EventVersion: 1,
			Timestamp:    time.Now(),
			Snapshot:     &snapshot,
		}
		repo.Create(event)
	}

	count, _ := reset.Execute()

	if count != 7 {
		t.Errorf("count = %d, want 7", count)
	}
}
