package usecases

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// TaskLookup resolves a task ID to a task (for getting parent/recur parent UUIDs)
type TaskLookup interface {
	Execute(id int64) (*task.Task, error)
}

// AreaLookup resolves an area ID to an area (for getting area UUID)
type AreaLookup interface {
	Execute(id int64) (*area.Area, error)
}

// PersistSyncEvent creates and stores sync events
type PersistSyncEvent struct {
	Repo       *syncevent.Repository
	TaskLookup TaskLookup
	AreaLookup AreaLookup
}

// PersistOptions contains the options for persisting a sync event
type PersistOptions struct {
	ClientID   string              // Required: originating client identifier
	EventType  syncevent.EventType // Required: type of event
	Task       *task.Task          // For task events
	Area       *area.Area          // For area events
	EntityUUID string              // Required for delete (when entity is already gone)
}

// Execute creates and persists a sync event for a task or area
func (p *PersistSyncEvent) Execute(opts *PersistOptions) (*syncevent.SyncEvent, error) {
	var entityType syncevent.EntityType
	var entityUUID string
	var snapshotJSON *string
	var entityTitle, entityStatus *string

	switch {
	case opts.Task != nil:
		entityType = syncevent.EntityTypeTask
		entityUUID = opts.Task.UUID
		entityTitle = &opts.Task.Title
		status := string(opts.Task.Status)
		entityStatus = &status

		// Build snapshot for all events (including deletes — needed for
		// entitySortOrder to distinguish project deletes from task deletes,
		// which is required for correct FK-safe delete ordering)
		snapshot := p.buildTaskSnapshot(opts.Task)
		jsonBytes, err := json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		jsonStr := string(jsonBytes)
		snapshotJSON = &jsonStr

	case opts.Area != nil:
		entityType = syncevent.EntityTypeArea
		entityUUID = opts.Area.UUID
		entityTitle = &opts.Area.Name
		// entityStatus stays nil for areas

		// Build snapshot for all events (including deletes)
		snapshot := &syncevent.AreaSnapshotData{
			UUID: opts.Area.UUID,
			Name: opts.Area.Name,
		}
		jsonBytes, err := json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		jsonStr := string(jsonBytes)
		snapshotJSON = &jsonStr

	default:
		// Fallback for delete by UUID (legacy support)
		entityType = syncevent.EntityTypeTask
		entityUUID = opts.EntityUUID
	}

	// Get next event version for this entity
	version, err := p.Repo.GetNextEventVersion(entityType, entityUUID)
	if err != nil {
		return nil, err
	}

	// Create the event
	event := &syncevent.SyncEvent{
		EventUUID:    uuid.New().String(),
		EntityType:   entityType,
		EntityUUID:   entityUUID,
		ClientID:     opts.ClientID,
		EventType:    opts.EventType,
		EventVersion: version,
		Timestamp:    time.Now(),
		Snapshot:     snapshotJSON,
		EntityTitle:  entityTitle,
		EntityStatus: entityStatus,
	}

	// Persist
	if err := p.Repo.Create(event); err != nil {
		return nil, err
	}

	return event, nil
}

// buildTaskSnapshot creates a TaskSnapshotData from a task
func (p *PersistSyncEvent) buildTaskSnapshot(t *task.Task) *syncevent.TaskSnapshotData {
	snapshot := &syncevent.TaskSnapshotData{
		UUID:        t.UUID,
		Title:       t.Title,
		Description: t.Description,
		TaskType:    string(t.TaskType),
		State:       string(t.State),
		Status:      string(t.Status),
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		RecurType:   t.RecurType,
		RecurRule:   t.RecurRule,
		RecurPaused: t.RecurPaused,
		Tags:        t.Tags,
	}

	// Resolve ParentID to ParentUUID
	if t.ParentID != nil {
		parent, err := p.TaskLookup.Execute(*t.ParentID)
		if err == nil && parent != nil {
			snapshot.ParentUUID = &parent.UUID
		}
	}

	// Resolve AreaID to AreaUUID
	if t.AreaID != nil {
		a, err := p.AreaLookup.Execute(*t.AreaID)
		if err == nil && a != nil {
			snapshot.AreaUUID = &a.UUID
		}
	}

	// Format dates
	if t.PlannedDate != nil {
		s := t.PlannedDate.Format("2006-01-02")
		snapshot.PlannedDate = &s
	}
	if t.DueDate != nil {
		s := t.DueDate.Format("2006-01-02")
		snapshot.DueDate = &s
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.Format(time.RFC3339)
		snapshot.CompletedAt = &s
	}
	if t.RecurEnd != nil {
		s := t.RecurEnd.Format("2006-01-02")
		snapshot.RecurEnd = &s
	}

	// Resolve RecurParentID to UUID
	if t.RecurParentID != nil {
		recurParent, err := p.TaskLookup.Execute(*t.RecurParentID)
		if err == nil && recurParent != nil {
			snapshot.RecurParentUUID = &recurParent.UUID
		}
	}

	return snapshot
}
