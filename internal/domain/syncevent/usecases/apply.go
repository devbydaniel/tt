package usecases

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// ErrTaskNotFound is returned when a task is not found (for delete operations).
var ErrTaskNotFound = errors.New("task not found")

// TaskUpserter defines the interface for upserting tasks.
type TaskUpserter interface {
	Upsert(task *task.Task) error
	DeleteByUUID(uuid string) error
}

// TaskByUUIDLookup resolves a task UUID to a task (for getting parent/recur parent IDs).
type TaskByUUIDLookup interface {
	GetByUUID(uuid string) (*task.Task, error)
}

// AreaByUUIDLookup resolves an area UUID to an area (for getting area ID).
type AreaByUUIDLookup interface {
	GetByUUID(uuid string) (*area.Area, error)
}

// AreaUpserter defines the interface for upserting areas.
type AreaUpserter interface {
	Upsert(a *area.Area) error
	DeleteByUUID(uuid string) error
}

// ApplyEntityStates applies entity states received from the server.
type ApplyEntityStates struct {
	TaskUpserter     TaskUpserter
	TaskByUUIDLookup TaskByUUIDLookup
	AreaByUUIDLookup AreaByUUIDLookup
	AreaUpserter     AreaUpserter
}

// ApplyResult contains the result of applying entity states.
type ApplyResult struct {
	Applied int
	Skipped int
}

// Apply applies the given entity states to the local database.
// Entities are sorted so areas are applied before tasks, ensuring
// that area references in tasks can be resolved.
func (a *ApplyEntityStates) Apply(entities []syncevent.EntityState) (*ApplyResult, error) {
	result := &ApplyResult{}

	// Sort entities: areas first, then tasks
	// This ensures areas exist before tasks that reference them are applied
	sort.SliceStable(entities, func(i, j int) bool {
		iIsArea := entities[i].EntityType == string(syncevent.EntityTypeArea)
		jIsArea := entities[j].EntityType == string(syncevent.EntityTypeArea)
		return iIsArea && !jIsArea
	})

	for _, entity := range entities {
		switch syncevent.EntityType(entity.EntityType) {
		case syncevent.EntityTypeTask:
			if err := a.applyTask(entity); err != nil {
				// Ignore "not found" errors for deletes - task may not exist locally
				if errors.Is(err, task.ErrTaskNotFound) {
					continue
				}
				return nil, err
			}
			result.Applied++

		case syncevent.EntityTypeArea:
			if err := a.applyArea(entity); err != nil {
				// Ignore "not found" errors for deletes - area may not exist locally
				if errors.Is(err, area.ErrAreaNotFound) {
					continue
				}
				return nil, err
			}
			result.Applied++

		default:
			result.Skipped++
		}
	}

	return result, nil
}

const dateFormat = "2006-01-02"

// applyTask applies a task entity state.
func (a *ApplyEntityStates) applyTask(entity syncevent.EntityState) error {
	eventType := syncevent.EventType(entity.EventType)

	// Handle deletes
	if eventType == syncevent.EventTypeDeleted {
		return a.TaskUpserter.DeleteByUUID(entity.EntityUUID)
	}

	// For created/updated/completed, parse snapshot and upsert
	if entity.Snapshot == nil {
		// No snapshot, nothing to apply
		return nil
	}

	var snapshot syncevent.TaskSnapshotData
	if err := json.Unmarshal([]byte(*entity.Snapshot), &snapshot); err != nil {
		return err
	}

	// Convert snapshot to task
	t, err := a.snapshotToTask(&snapshot)
	if err != nil {
		return err
	}

	return a.TaskUpserter.Upsert(t)
}

// snapshotToTask converts a TaskSnapshotData to a Task, resolving UUIDs/names to local IDs.
func (a *ApplyEntityStates) snapshotToTask(snapshot *syncevent.TaskSnapshotData) (*task.Task, error) {
	// Parse dates
	var plannedDate, dueDate, recurEnd *time.Time
	if snapshot.PlannedDate != nil {
		parsed, _ := time.Parse(dateFormat, *snapshot.PlannedDate)
		plannedDate = &parsed
	}
	if snapshot.DueDate != nil {
		parsed, _ := time.Parse(dateFormat, *snapshot.DueDate)
		dueDate = &parsed
	}
	if snapshot.RecurEnd != nil {
		parsed, _ := time.Parse(dateFormat, *snapshot.RecurEnd)
		recurEnd = &parsed
	}

	var createdAt time.Time
	if snapshot.CreatedAt != "" {
		createdAt, _ = time.Parse(time.RFC3339, snapshot.CreatedAt)
	} else {
		createdAt = time.Now()
	}

	var completedAt *time.Time
	if snapshot.CompletedAt != nil {
		parsed, _ := time.Parse(time.RFC3339, *snapshot.CompletedAt)
		completedAt = &parsed
	}

	// Resolve parent UUID to ID
	var parentID *int64
	if snapshot.ParentUUID != nil && a.TaskByUUIDLookup != nil {
		parent, err := a.TaskByUUIDLookup.GetByUUID(*snapshot.ParentUUID)
		if err == nil {
			parentID = &parent.ID
		}
		// If parent not found, leave parentID nil
	}

	// Resolve area UUID to ID
	var areaID *int64
	if snapshot.AreaUUID != nil && a.AreaByUUIDLookup != nil {
		ar, err := a.AreaByUUIDLookup.GetByUUID(*snapshot.AreaUUID)
		if err == nil {
			areaID = &ar.ID
		}
		// If area not found, leave areaID nil
	}

	// Resolve recur parent UUID to ID
	var recurParentID *int64
	if snapshot.RecurParentUUID != nil && a.TaskByUUIDLookup != nil {
		recurParent, err := a.TaskByUUIDLookup.GetByUUID(*snapshot.RecurParentUUID)
		if err == nil {
			recurParentID = &recurParent.ID
		}
	}

	return &task.Task{
		UUID:          snapshot.UUID,
		Title:         snapshot.Title,
		Description:   snapshot.Description,
		TaskType:      task.TaskType(snapshot.TaskType),
		ParentID:      parentID,
		AreaID:        areaID,
		PlannedDate:   plannedDate,
		DueDate:       dueDate,
		State:         task.State(snapshot.State),
		Status:        task.Status(snapshot.Status),
		CreatedAt:     createdAt,
		CompletedAt:   completedAt,
		RecurType:     snapshot.RecurType,
		RecurRule:     snapshot.RecurRule,
		RecurEnd:      recurEnd,
		RecurPaused:   snapshot.RecurPaused,
		RecurParentID: recurParentID,
		Tags:          snapshot.Tags,
	}, nil
}

// applyArea applies an area entity state.
func (a *ApplyEntityStates) applyArea(entity syncevent.EntityState) error {
	eventType := syncevent.EventType(entity.EventType)

	// Handle deletes
	if eventType == syncevent.EventTypeDeleted {
		return a.AreaUpserter.DeleteByUUID(entity.EntityUUID)
	}

	// For created/updated, parse snapshot and upsert
	if entity.Snapshot == nil {
		// No snapshot, nothing to apply
		return nil
	}

	var snapshot syncevent.AreaSnapshotData
	if err := json.Unmarshal([]byte(*entity.Snapshot), &snapshot); err != nil {
		return err
	}

	ar := &area.Area{
		UUID: snapshot.UUID,
		Name: snapshot.Name,
	}

	return a.AreaUpserter.Upsert(ar)
}
