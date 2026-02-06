package usecases_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/testutil"
)

// mockApplier implements usecases.EntityApplier
type mockApplier struct {
	appliedEntities []syncevent.EntityState
	applyErr        error
}

func (m *mockApplier) Apply(entities []syncevent.EntityState) (*usecases.ApplyResult, error) {
	if m.applyErr != nil {
		return nil, m.applyErr
	}
	m.appliedEntities = append(m.appliedEntities, entities...)
	return &usecases.ApplyResult{Applied: len(entities)}, nil
}

func setupSyncTest(t *testing.T, handler http.HandlerFunc) (*usecases.SyncEvents, *syncevent.Repository, *mockApplier, *httptest.Server) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := syncevent.NewRepository(db)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := syncevent.NewClient(server.URL, "test-api-key")
	applier := &mockApplier{}

	syncEvents := &usecases.SyncEvents{
		Repo:     repo,
		Client:   client,
		ClientID: "test-client",
		Applier:  applier,
	}

	return syncEvents, repo, applier, server
}

func createUnpushedEvent(repo *syncevent.Repository, entityUUID, eventUUID string) {
	snapshot := `{"uuid":"` + entityUUID + `","title":"Test","taskType":"task","state":"inbox","status":"todo","createdAt":"2024-01-01T00:00:00Z"}`
	event := &syncevent.SyncEvent{
		EventUUID:    eventUUID,
		EntityType:   syncevent.EntityTypeTask,
		EntityUUID:   entityUUID,
		ClientID:     "test-client",
		EventType:    syncevent.EventTypeCreated,
		EventVersion: 1,
		Timestamp:    time.Now(),
		Snapshot:     &snapshot,
	}
	repo.Create(event)
}

func TestSyncPushesUnpushedEvents(t *testing.T) {
	pushedEvents := make([]*syncevent.SyncEvent, 0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.SyncRequest
		json.NewDecoder(r.Body).Decode(&req)

		pushedEvents = append(pushedEvents, req.Events...)

		// Accept all events
		accepted := make([]string, len(req.Events))
		for i, e := range req.Events {
			accepted[i] = e.EventUUID
		}

		resp := syncevent.SyncResponse{
			Accepted:  accepted,
			Rejected:  []syncevent.RejectedEvent{},
			Entities:  []syncevent.EntityState{},
			NewCursor: 10,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)

	// Create unpushed events
	createUnpushedEvent(repo, "entity-1", "event-1")
	createUnpushedEvent(repo, "entity-2", "event-2")

	result, err := sync.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Pushed != 2 {
		t.Errorf("pushed = %d, want 2", result.Pushed)
	}
	if len(pushedEvents) != 2 {
		t.Errorf("server received %d events, want 2", len(pushedEvents))
	}
}

func TestSyncMarksEventsAsPushed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.SyncRequest
		json.NewDecoder(r.Body).Decode(&req)

		accepted := make([]string, len(req.Events))
		for i, e := range req.Events {
			accepted[i] = e.EventUUID
		}

		resp := syncevent.SyncResponse{
			Accepted:  accepted,
			NewCursor: 10,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)

	createUnpushedEvent(repo, "entity-1", "event-1")

	// Before sync, 1 unpushed
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 1 {
		t.Fatalf("should have 1 unpushed before sync")
	}

	sync.Execute()

	// After sync, 0 unpushed
	unpushed, _ = repo.GetUnpushed(10)
	if len(unpushed) != 0 {
		t.Errorf("should have 0 unpushed after sync, got %d", len(unpushed))
	}
}

func TestSyncRecordsRejectedEvents(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := syncevent.SyncResponse{
			Accepted: []string{},
			Rejected: []syncevent.RejectedEvent{
				{EventUUID: "event-1", Reason: "duplicate"},
			},
			NewCursor: 10,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)
	createUnpushedEvent(repo, "entity-1", "event-1")

	result, _ := sync.Execute()

	if result.Rejected != 1 {
		t.Errorf("rejected = %d, want 1", result.Rejected)
	}
	if len(result.Errors) != 1 {
		t.Errorf("errors count = %d, want 1", len(result.Errors))
	}
	if result.Errors[0] != "event-1: duplicate" {
		t.Errorf("error message = %s, want 'event-1: duplicate'", result.Errors[0])
	}
}

func TestSyncAppliesReceivedEntities(t *testing.T) {
	snapshot := makeSnapshot("remote-task", "Task from server")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := syncevent.SyncResponse{
			Accepted: []string{},
			Entities: []syncevent.EntityState{
				{
					EntityType: string(syncevent.EntityTypeTask),
					EntityUUID: "remote-task",
					EventType:  string(syncevent.EventTypeCreated),
					Snapshot:   &snapshot,
				},
			},
			NewCursor: 10,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, _, applier, _ := setupSyncTest(t, handler)

	result, err := sync.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Pulled != 1 {
		t.Errorf("pulled = %d, want 1", result.Pulled)
	}
	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1", result.Applied)
	}
	if len(applier.appliedEntities) != 1 {
		t.Errorf("applier received %d entities, want 1", len(applier.appliedEntities))
	}
}

func TestSyncUpdatesCursor(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := syncevent.SyncResponse{
			Accepted:  []string{},
			NewCursor: 42,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)

	sync.Execute()

	cursor, _ := repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "42" {
		t.Errorf("cursor = %s, want 42", cursor)
	}
}

func TestSyncUsesCursorFromState(t *testing.T) {
	receivedCursor := int64(0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.SyncRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedCursor = req.Cursor

		resp := syncevent.SyncResponse{
			Accepted:  []string{},
			NewCursor: req.Cursor + 10,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)

	// Set initial cursor
	repo.SetSyncState(usecases.SyncStateServerCursor, "100")

	sync.Execute()

	if receivedCursor != 100 {
		t.Errorf("server received cursor = %d, want 100", receivedCursor)
	}
}

func TestSyncBatching(t *testing.T) {
	requestCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req syncevent.SyncRequest
		json.NewDecoder(r.Body).Decode(&req)

		accepted := make([]string, len(req.Events))
		for i, e := range req.Events {
			accepted[i] = e.EventUUID
		}

		resp := syncevent.SyncResponse{
			Accepted:  accepted,
			NewCursor: int64(requestCount * 10),
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)

	// Create more events than default batch size (100)
	// We'll create just 3 to test the "less than batch size" termination
	for i := 0; i < 3; i++ {
		snapshot := `{"uuid":"entity-` + string(rune('0'+i)) + `","title":"Test","taskType":"task","state":"inbox","status":"todo","createdAt":"2024-01-01T00:00:00Z"}`
		event := &syncevent.SyncEvent{
			EventUUID:    "event-" + string(rune('0'+i)),
			EntityType:   syncevent.EntityTypeTask,
			EntityUUID:   "entity-" + string(rune('0'+i)),
			ClientID:     "test-client",
			EventType:    syncevent.EventTypeCreated,
			EventVersion: 1,
			Timestamp:    time.Now(),
			Snapshot:     &snapshot,
		}
		repo.Create(event)
	}

	result, _ := sync.Execute()

	// With 3 events (less than batch size of 100), should make exactly 1 request
	if requestCount != 1 {
		t.Errorf("request count = %d, want 1 (events < batch size)", requestCount)
	}
	if result.Pushed != 3 {
		t.Errorf("pushed = %d, want 3", result.Pushed)
	}
}

func TestSyncEmptyQueue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := syncevent.SyncResponse{
			Accepted:  []string{},
			NewCursor: 5,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, _, _, _ := setupSyncTest(t, handler)

	result, err := sync.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Pushed != 0 {
		t.Errorf("pushed = %d, want 0", result.Pushed)
	}
	if result.Pulled != 0 {
		t.Errorf("pulled = %d, want 0", result.Pulled)
	}
}

func TestSyncHandlesServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)
	createUnpushedEvent(repo, "entity-1", "event-1")

	_, err := sync.Execute()
	if err == nil {
		t.Error("Execute() should return error on server failure")
	}
}

func TestSyncHandlesAuthError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)
	createUnpushedEvent(repo, "entity-1", "event-1")

	_, err := sync.Execute()
	if err == nil {
		t.Error("Execute() should return error on auth failure")
	}
}

func TestSyncCursorOnlyUpdatesWhenHigher(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := syncevent.SyncResponse{
			Accepted:  []string{},
			NewCursor: 5, // Lower than initial cursor
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)
	repo.SetSyncState(usecases.SyncStateServerCursor, "100")

	sync.Execute()

	// Cursor should remain at 100 (not downgraded to 5)
	cursor, _ := repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "100" {
		t.Errorf("cursor = %s, want 100 (should not downgrade)", cursor)
	}
}

func TestSyncRejectedEventsGetFailureTracking(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.SyncRequest
		json.NewDecoder(r.Body).Decode(&req)

		rejected := make([]syncevent.RejectedEvent, len(req.Events))
		for i, e := range req.Events {
			rejected[i] = syncevent.RejectedEvent{
				EventUUID: e.EventUUID,
				Reason:    "bad client id",
			}
		}

		resp := syncevent.SyncResponse{
			Accepted:  []string{},
			Rejected:  rejected,
			NewCursor: 10,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, _, _ := setupSyncTest(t, handler)
	createUnpushedEvent(repo, "entity-1", "event-1")

	// Reject MaxFailureCount times
	for i := 0; i < syncevent.MaxFailureCount; i++ {
		sync.Execute()
	}

	// Event should be permanently failed
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 0 {
		t.Errorf("should have 0 unpushed after %d rejections, got %d", syncevent.MaxFailureCount, len(unpushed))
	}

	failed, _ := repo.GetPermanentlyFailed()
	if len(failed) != 1 {
		t.Errorf("should have 1 permanently failed, got %d", len(failed))
	}
}

func TestSyncFullCycle(t *testing.T) {
	// Simulates a realistic sync cycle:
	// 1. Client has 2 local changes to push
	// 2. Server accepts them
	// 3. Server returns 1 remote change to apply
	// 4. Client applies the change

	remoteSnapshot := makeSnapshot("remote-task", "Task from other device")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.SyncRequest
		json.NewDecoder(r.Body).Decode(&req)

		accepted := make([]string, len(req.Events))
		for i, e := range req.Events {
			accepted[i] = e.EventUUID
		}

		resp := syncevent.SyncResponse{
			Accepted: accepted,
			Rejected: []syncevent.RejectedEvent{},
			Entities: []syncevent.EntityState{
				{
					EntityType: string(syncevent.EntityTypeTask),
					EntityUUID: "remote-task",
					EventType:  string(syncevent.EventTypeCreated),
					Snapshot:   &remoteSnapshot,
				},
			},
			NewCursor: 50,
		}
		json.NewEncoder(w).Encode(resp)
	})

	sync, repo, applier, _ := setupSyncTest(t, handler)

	// Create local changes
	createUnpushedEvent(repo, "local-task-1", "event-1")
	createUnpushedEvent(repo, "local-task-2", "event-2")

	result, err := sync.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify results
	if result.Pushed != 2 {
		t.Errorf("pushed = %d, want 2", result.Pushed)
	}
	if result.Pulled != 1 {
		t.Errorf("pulled = %d, want 1", result.Pulled)
	}
	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1", result.Applied)
	}
	if result.Rejected != 0 {
		t.Errorf("rejected = %d, want 0", result.Rejected)
	}

	// Local events should be marked as pushed
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 0 {
		t.Errorf("unpushed after sync = %d, want 0", len(unpushed))
	}

	// Cursor should be updated
	cursor, _ := repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "50" {
		t.Errorf("cursor = %s, want 50", cursor)
	}

	// Remote entity should be applied
	if len(applier.appliedEntities) != 1 {
		t.Errorf("applied entities = %d, want 1", len(applier.appliedEntities))
	}
	if applier.appliedEntities[0].EntityUUID != "remote-task" {
		t.Errorf("applied entity UUID = %s, want remote-task", applier.appliedEntities[0].EntityUUID)
	}
}
