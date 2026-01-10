package usecases

import (
	"encoding/json"
	"time"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/google/uuid"
)

// TaskLookup resolves a task ID to a task (for getting parent/recur parent UUIDs)
type TaskLookup interface {
	Execute(id int64) (*task.Task, error)
}

// AreaLookup resolves an area ID to an area (for getting area name)
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
	ClientID   string               // Required: originating client identifier
	EventType  syncevent.EventType  // Required: type of event
	Task       *task.Task           // Required for create/update/complete; nil for delete
	EntityUUID string               // Required for delete (when task is already gone)
}

// Execute creates and persists a sync event for a task
func (p *PersistSyncEvent) Execute(opts *PersistOptions) (*syncevent.SyncEvent, error) {
	// Determine entity UUID
	entityUUID := opts.EntityUUID
	if opts.Task != nil {
		entityUUID = opts.Task.UUID
	}

	// Get next event version for this entity
	version, err := p.Repo.GetNextEventVersion(syncevent.EntityTypeTask, entityUUID)
	if err != nil {
		return nil, err
	}

	// Build snapshot (nil for deletes) and extract metadata
	var snapshotJSON *string
	var entityTitle, entityStatus *string

	if opts.Task != nil {
		// Always capture title for reference
		entityTitle = &opts.Task.Title
		status := string(opts.Task.Status)
		entityStatus = &status

		// Build full snapshot for non-delete events
		if opts.EventType != syncevent.EventTypeDeleted {
			snapshot, err := p.buildTaskSnapshot(opts.Task)
			if err != nil {
				return nil, err
			}
			jsonBytes, err := json.Marshal(snapshot)
			if err != nil {
				return nil, err
			}
			jsonStr := string(jsonBytes)
			snapshotJSON = &jsonStr
		}
	}

	// Create the event
	event := &syncevent.SyncEvent{
		EventUUID:    uuid.New().String(),
		EntityType:   syncevent.EntityTypeTask,
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
func (p *PersistSyncEvent) buildTaskSnapshot(t *task.Task) (*syncevent.TaskSnapshotData, error) {
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

	// Resolve AreaID to AreaName
	if t.AreaID != nil {
		a, err := p.AreaLookup.Execute(*t.AreaID)
		if err == nil && a != nil {
			snapshot.AreaName = &a.Name
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

	return snapshot, nil
}
