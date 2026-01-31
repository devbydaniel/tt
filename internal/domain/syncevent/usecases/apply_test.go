package usecases_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// mockTaskUpserter implements usecases.TaskUpserter for testing
type mockTaskUpserter struct {
	upserted   []*task.Task
	deleted    []string
	upsertErr  error
	deleteErr  error
	notFoundOn string // Return ErrTaskNotFound for this UUID
}

func (m *mockTaskUpserter) Upsert(t *task.Task) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.upserted = append(m.upserted, t)
	return nil
}

func (m *mockTaskUpserter) DeleteByUUID(uuid string) error {
	if uuid == m.notFoundOn {
		return task.ErrTaskNotFound
	}
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, uuid)
	return nil
}

func makeSnapshot(uuid, title string) string {
	data := syncevent.TaskSnapshotData{
		UUID:      uuid,
		Title:     title,
		TaskType:  "task",
		State:     "inbox",
		Status:    "todo",
		CreatedAt: "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func TestApplyCreatedEntity(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	snapshot := makeSnapshot("task-1", "New Task")
	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeCreated),
			Snapshot:   &snapshot,
		},
	}

	result, err := apply.Apply(entities)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1", result.Applied)
	}
	if len(mock.upserted) != 1 {
		t.Errorf("upserted count = %d, want 1", len(mock.upserted))
	}
	if mock.upserted[0].UUID != "task-1" {
		t.Errorf("upserted UUID = %s, want task-1", mock.upserted[0].UUID)
	}
	if mock.upserted[0].Title != "New Task" {
		t.Errorf("upserted title = %s, want 'New Task'", mock.upserted[0].Title)
	}
}

func TestApplyUpdatedEntity(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	snapshot := makeSnapshot("task-1", "Updated Task")
	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeUpdated),
			Snapshot:   &snapshot,
		},
	}

	result, err := apply.Apply(entities)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1", result.Applied)
	}
	if mock.upserted[0].Title != "Updated Task" {
		t.Errorf("upserted title = %s, want 'Updated Task'", mock.upserted[0].Title)
	}
}

func TestApplyCompletedEntity(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	snapshot := makeSnapshot("task-1", "Completed Task")
	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeCompleted),
			Snapshot:   &snapshot,
		},
	}

	result, _ := apply.Apply(entities)

	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1", result.Applied)
	}
}

func TestApplyDeletedEntity(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-to-delete",
			EventType:  string(syncevent.EventTypeDeleted),
			Snapshot:   nil, // Deletes have no snapshot
		},
	}

	result, err := apply.Apply(entities)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1", result.Applied)
	}
	if len(mock.deleted) != 1 {
		t.Errorf("deleted count = %d, want 1", len(mock.deleted))
	}
	if mock.deleted[0] != "task-to-delete" {
		t.Errorf("deleted UUID = %s, want task-to-delete", mock.deleted[0])
	}
	if len(mock.upserted) != 0 {
		t.Error("upsert should not be called for deletes")
	}
}

func TestApplyDeleteIgnoresNotFound(t *testing.T) {
	mock := &mockTaskUpserter{notFoundOn: "nonexistent-task"}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "nonexistent-task",
			EventType:  string(syncevent.EventTypeDeleted),
			Snapshot:   nil,
		},
	}

	// Should not error - just skips silently
	result, err := apply.Apply(entities)
	if err != nil {
		t.Errorf("Apply() should not error for not found delete, got %v", err)
	}

	// Not found deletes count as 0 applied (continue past them)
	if result.Applied != 0 {
		t.Errorf("applied = %d, want 0 (task not found)", result.Applied)
	}
}

func TestApplyMultipleEntities(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	snapshot1 := makeSnapshot("task-1", "Task 1")
	snapshot2 := makeSnapshot("task-2", "Task 2")

	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeCreated),
			Snapshot:   &snapshot1,
		},
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-2",
			EventType:  string(syncevent.EventTypeCreated),
			Snapshot:   &snapshot2,
		},
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-3",
			EventType:  string(syncevent.EventTypeDeleted),
			Snapshot:   nil,
		},
	}

	result, _ := apply.Apply(entities)

	if result.Applied != 3 {
		t.Errorf("applied = %d, want 3", result.Applied)
	}
	if len(mock.upserted) != 2 {
		t.Errorf("upserted count = %d, want 2", len(mock.upserted))
	}
	if len(mock.deleted) != 1 {
		t.Errorf("deleted count = %d, want 1", len(mock.deleted))
	}
}

func TestApplySkipsUnknownEntityTypes(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	snapshot := makeSnapshot("task-1", "Task")
	entities := []syncevent.EntityState{
		{
			EntityType: "unknown_type",
			EntityUUID: "unknown-1",
			EventType:  string(syncevent.EventTypeCreated),
			Snapshot:   &snapshot,
		},
	}

	result, err := apply.Apply(entities)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Applied != 0 {
		t.Errorf("applied = %d, want 0", result.Applied)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
}

func TestApplyNilSnapshotNonDelete(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	// Non-delete event with nil snapshot - gracefully skips without error
	// Since no error occurs, the entity is counted as "processed" even though
	// nothing was actually written to the database
	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeUpdated),
			Snapshot:   nil, // No snapshot to apply
		},
	}

	result, err := apply.Apply(entities)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Counted as processed (no error) even though nothing written
	if result.Applied != 1 {
		t.Errorf("applied = %d, want 1 (processed without error)", result.Applied)
	}

	// But TaskUpserter should NOT have been called
	if len(mock.upserted) != 0 {
		t.Error("upsert should not be called when snapshot is nil")
	}
}

func TestApplyPropagatesUpsertError(t *testing.T) {
	mock := &mockTaskUpserter{upsertErr: errors.New("database error")}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	snapshot := makeSnapshot("task-1", "Task")
	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeCreated),
			Snapshot:   &snapshot,
		},
	}

	_, err := apply.Apply(entities)
	if err == nil {
		t.Error("Apply() should propagate upsert error")
	}
}

func TestApplyPropagatesDeleteError(t *testing.T) {
	mock := &mockTaskUpserter{deleteErr: errors.New("database error")}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeDeleted),
			Snapshot:   nil,
		},
	}

	_, err := apply.Apply(entities)
	if err == nil {
		t.Error("Apply() should propagate delete error")
	}
}

func TestApplyInvalidJSON(t *testing.T) {
	mock := &mockTaskUpserter{}
	apply := &usecases.ApplyEntityStates{TaskUpserter: mock}

	invalidJSON := "not valid json"
	entities := []syncevent.EntityState{
		{
			EntityType: string(syncevent.EntityTypeTask),
			EntityUUID: "task-1",
			EventType:  string(syncevent.EventTypeCreated),
			Snapshot:   &invalidJSON,
		},
	}

	_, err := apply.Apply(entities)
	if err == nil {
		t.Error("Apply() should error on invalid JSON snapshot")
	}
}
