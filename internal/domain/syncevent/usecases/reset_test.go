package usecases_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/testutil"
)

// --- ResetSyncEvents tests (server-side, simple reset) ---

func setupResetSyncEvents(t *testing.T) (*usecases.ResetSyncEvents, *syncevent.Repository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := syncevent.NewRepository(db)
	return &usecases.ResetSyncEvents{Repo: repo}, repo
}

func TestResetSyncEventsDeletesAllEvents(t *testing.T) {
	reset, repo := setupResetSyncEvents(t)

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

	result, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.DeletedEvents != 5 {
		t.Errorf("deleted count = %d, want 5", result.DeletedEvents)
	}

	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 0 {
		t.Errorf("should have 0 events after reset, got %d", len(unpushed))
	}
}

func TestResetSyncEventsClearsCursor(t *testing.T) {
	reset, repo := setupResetSyncEvents(t)

	repo.SetSyncState(usecases.SyncStateServerCursor, "12345")

	reset.Execute()

	cursor, _ := repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "0" {
		t.Errorf("cursor = %s, want '0'", cursor)
	}
}

func TestResetSyncEventsEmptyDatabase(t *testing.T) {
	reset, _ := setupResetSyncEvents(t)

	result, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.DeletedEvents != 0 {
		t.Errorf("deleted count = %d, want 0", result.DeletedEvents)
	}
}

// --- ResetSync tests (client-side, full reset) ---

func setupResetSync(t *testing.T, serverHandler http.HandlerFunc) (*usecases.ResetSync, *syncevent.Repository, *task.Repository, *area.Repository, *database.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := syncevent.NewRepository(db)
	taskRepo := task.NewRepository(db)
	areaRepo := area.NewRepository(db)

	server := httptest.NewServer(serverHandler)
	t.Cleanup(server.Close)

	client := syncevent.NewClient(server.URL, "test-api-key")

	getTask := &taskLookup{repo: taskRepo}
	getArea := &areaLookup{repo: areaRepo}

	persister := &usecases.PersistSyncEvent{
		Repo:       repo,
		TaskLookup: getTask,
		AreaLookup: getArea,
	}

	listAllTasks := &listAllTasksUC{repo: taskRepo}
	listAllAreas := &listAllAreasUC{repo: areaRepo}

	reset := &usecases.ResetSync{
		Repo:          repo,
		Client:        client,
		TaskLister:    listAllTasks,
		AreaLister:    listAllAreas,
		SyncPersister: persister,
		ClientID:      "test-client",
		DB:            db,
	}

	return reset, repo, taskRepo, areaRepo, db
}

// listAllTasksUC implements usecases.AllTasksLister
type listAllTasksUC struct {
	repo *task.Repository
}

func (l *listAllTasksUC) Execute() ([]task.Task, error) {
	return l.repo.ListAll()
}

// listAllAreasUC implements usecases.AllAreasLister
type listAllAreasUC struct {
	repo *area.Repository
}

func (l *listAllAreasUC) Execute() ([]area.Area, error) {
	return l.repo.List()
}

// taskLookup implements usecases.TaskLookup
type taskLookup struct {
	repo *task.Repository
}

func (l *taskLookup) Execute(id int64) (*task.Task, error) {
	return l.repo.GetByID(id)
}

// areaLookup implements usecases.AreaLookup
type areaLookup struct {
	repo *area.Repository
}

func (l *areaLookup) Execute(id int64) (*area.Area, error) {
	return l.repo.GetByID(id)
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"deletedEvents": 0, "deletedTasks": 0, "deletedAreas": 0})
}

func TestResetSyncResetsServerFirst(t *testing.T) {
	serverCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		if r.URL.Path != "/api/v1/sync/reset" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		okHandler(w, r)
	})

	reset, _, _, _, _ := setupResetSync(t, handler)

	_, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !serverCalled {
		t.Error("server reset endpoint was not called")
	}
}

func TestResetSyncFailsIfServerUnreachable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	reset, repo, _, _, _ := setupResetSync(t, handler)

	// Create an event to verify it's NOT deleted on failure
	snapshot := `{}`
	repo.Create(&syncevent.SyncEvent{
		EventUUID: "event-1", EntityType: syncevent.EntityTypeTask, EntityUUID: "entity-1",
		ClientID: "test-client", EventType: syncevent.EventTypeCreated, EventVersion: 1,
		Timestamp: time.Now(), Snapshot: &snapshot,
	})

	_, err := reset.Execute()
	if err == nil {
		t.Fatal("Execute() should fail when server returns error")
	}

	// Local events should NOT be deleted
	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 1 {
		t.Errorf("local events should be preserved on server failure, got %d", len(unpushed))
	}
}

func TestResetSyncRegeneratesEventsForTasks(t *testing.T) {
	reset, repo, taskRepo, _, _ := setupResetSync(t, http.HandlerFunc(okHandler))

	// Create some tasks directly in the DB
	taskRepo.Create(&task.Task{UUID: "uuid-1", Title: "Task 1", TaskType: task.TaskTypeTask, State: task.StateActive, Status: task.StatusTodo})
	taskRepo.Create(&task.Task{UUID: "uuid-2", Title: "Task 2", TaskType: task.TaskTypeTask, State: task.StateActive, Status: task.StatusTodo})

	result, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.RegeneratedEvents != 2 {
		t.Errorf("regenerated = %d, want 2", result.RegeneratedEvents)
	}

	// Verify events exist in repo
	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 2 {
		t.Errorf("should have 2 unpushed events, got %d", len(unpushed))
	}
}

func TestResetSyncRegeneratesEventsForAreas(t *testing.T) {
	reset, repo, _, areaRepo, _ := setupResetSync(t, http.HandlerFunc(okHandler))

	areaRepo.Create(&area.Area{Name: "Work"})
	areaRepo.Create(&area.Area{Name: "Personal"})

	result, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.RegeneratedEvents != 2 {
		t.Errorf("regenerated = %d, want 2", result.RegeneratedEvents)
	}

	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 2 {
		t.Errorf("should have 2 unpushed events, got %d", len(unpushed))
	}
}

func TestResetSyncClearsOldEventsBeforeRegenerating(t *testing.T) {
	reset, repo, taskRepo, _, _ := setupResetSync(t, http.HandlerFunc(okHandler))

	// Create old sync events
	snapshot := `{}`
	for i := 0; i < 5; i++ {
		repo.Create(&syncevent.SyncEvent{
			EventUUID: "old-event-" + string(rune('0'+i)), EntityType: syncevent.EntityTypeTask,
			EntityUUID: "old-entity-" + string(rune('0'+i)), ClientID: "test-client",
			EventType: syncevent.EventTypeCreated, EventVersion: 1,
			Timestamp: time.Now(), Snapshot: &snapshot,
		})
	}

	// Create one task
	taskRepo.Create(&task.Task{UUID: "uuid-1", Title: "Task 1", TaskType: task.TaskTypeTask, State: task.StateActive, Status: task.StatusTodo})

	result, err := reset.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.DeletedEvents != 5 {
		t.Errorf("deleted = %d, want 5", result.DeletedEvents)
	}
	if result.RegeneratedEvents != 1 {
		t.Errorf("regenerated = %d, want 1", result.RegeneratedEvents)
	}

	// Only the regenerated event should exist
	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 1 {
		t.Errorf("should have 1 event after reset, got %d", len(unpushed))
	}
}

func TestResetSyncCompletedTasksGetCorrectEventType(t *testing.T) {
	reset, repo, taskRepo, _, _ := setupResetSync(t, http.HandlerFunc(okHandler))

	// Create a completed task
	completedAt := time.Now()
	taskRepo.Create(&task.Task{
		UUID: "uuid-done", Title: "Done task", TaskType: task.TaskTypeTask,
		State: task.StateActive, Status: task.StatusDone,
		CompletedAt: &completedAt,
	})

	reset.Execute()

	unpushed, _ := repo.GetUnpushed(100)
	if len(unpushed) != 1 {
		t.Fatalf("should have 1 event, got %d", len(unpushed))
	}

	if unpushed[0].EventType != syncevent.EventTypeCompleted {
		t.Errorf("event type = %s, want %s", unpushed[0].EventType, syncevent.EventTypeCompleted)
	}
}

func TestResetSyncResetsCursor(t *testing.T) {
	reset, repo, _, _, _ := setupResetSync(t, http.HandlerFunc(okHandler))

	repo.SetSyncState(usecases.SyncStateServerCursor, "999")

	reset.Execute()

	cursor, _ := repo.GetSyncState(usecases.SyncStateServerCursor)
	if cursor != "0" {
		t.Errorf("cursor = %s, want '0'", cursor)
	}
}
