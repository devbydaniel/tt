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

// PendingRepo defines the interface for managing the pending resolution queue.
type PendingRepo interface {
	SavePending(state syncevent.EntityState) error
	GetPending() ([]syncevent.EntityState, error)
	IncrementPendingRetry(entityUUID string) error
	RemovePending(entityUUID string) error
}

// ApplyEntityStates applies entity states received from the server.
type ApplyEntityStates struct {
	TaskUpserter     TaskUpserter
	TaskByUUIDLookup TaskByUUIDLookup
	AreaByUUIDLookup AreaByUUIDLookup
	AreaUpserter     AreaUpserter
	PendingRepo      PendingRepo // Optional: if set, enables retry queue for unresolved references
}

// ApplyResult contains the result of applying entity states.
type ApplyResult struct {
	Applied  int
	Skipped  int
	Deferred int // entities queued for later resolution
}

// entitySortOrder returns a numeric priority for sorting entities.
// For creates/updates: areas (0) → projects (1) → tasks (2) — parents before children.
// For deletes: tasks (0) → projects (1) → areas (2) — children before parents,
// so that FK constraints (ON DELETE RESTRICT) are not violated.
func entitySortOrder(e syncevent.EntityState) int {
	isDelete := syncevent.EventType(e.EventType) == syncevent.EventTypeDeleted

	var order int
	if e.EntityType == string(syncevent.EntityTypeArea) {
		order = 0
	} else if e.Snapshot != nil {
		// For tasks, peek at the snapshot to determine if it's a project
		var peek struct {
			TaskType string `json:"taskType"`
		}
		if json.Unmarshal([]byte(*e.Snapshot), &peek) == nil && peek.TaskType == "project" {
			order = 1
		} else {
			order = 2
		}
	} else {
		// No snapshot (typical for deletes) — treat as regular task
		order = 2
	}

	if isDelete {
		return 2 - order
	}
	return order
}

// Apply applies the given entity states to the local database.
// Entities are sorted so areas are applied first, then project tasks,
// then regular tasks — ensuring that references can be resolved.
//
// After the first pass, any tasks with unresolved references are re-applied
// (second pass) since the missing entities may now exist from the same batch.
// If references still can't be resolved and PendingRepo is set, the entities
// are queued for retry after future batches.
func (a *ApplyEntityStates) Apply(entities []syncevent.EntityState) (*ApplyResult, error) {
	result := &ApplyResult{}

	// Sort entities: areas first, then projects, then regular tasks
	// This ensures areas and projects exist before tasks that reference them
	sort.SliceStable(entities, func(i, j int) bool {
		return entitySortOrder(entities[i]) < entitySortOrder(entities[j])
	})

	// First pass: apply all entities, tracking which tasks had unresolved references
	var unresolvedTasks []syncevent.EntityState

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

			// Check if this task had unresolved references
			if a.taskHasUnresolvedRefs(entity) {
				unresolvedTasks = append(unresolvedTasks, entity)
			}

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

	// Second pass: re-apply tasks that had unresolved references.
	// Their parents/areas may now exist from the same batch.
	for i, entity := range unresolvedTasks {
		if err := a.applyTask(entity); err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				continue
			}
			return nil, err
		}
		// Check if still unresolved after second attempt
		if !a.taskHasUnresolvedRefs(entity) {
			// Resolved! Remove from list so we don't defer it
			unresolvedTasks[i].EntityUUID = "" // mark as resolved
		}
	}

	// Queue still-unresolved entities for future retry
	if a.PendingRepo != nil {
		for _, entity := range unresolvedTasks {
			if entity.EntityUUID == "" {
				continue // was resolved in second pass
			}
			if err := a.PendingRepo.SavePending(entity); err != nil {
				return nil, err
			}
			result.Deferred++
		}
	}

	// Retry previously pending entities that might now be resolvable
	if a.PendingRepo != nil {
		resolved, err := a.retryPending()
		if err != nil {
			return nil, err
		}
		result.Applied += resolved
	}

	return result, nil
}

// retryPending attempts to re-apply entities from the pending resolution queue.
// Returns the number of successfully resolved entities.
func (a *ApplyEntityStates) retryPending() (int, error) {
	pending, err := a.PendingRepo.GetPending()
	if err != nil {
		return 0, err
	}

	resolved := 0
	for _, entity := range pending {
		switch syncevent.EntityType(entity.EntityType) {
		case syncevent.EntityTypeTask:
			if err := a.applyTask(entity); err != nil {
				if errors.Is(err, task.ErrTaskNotFound) {
					continue
				}
				// Increment retry count on failure
				_ = a.PendingRepo.IncrementPendingRetry(entity.EntityUUID)
				continue
			}
			if !a.taskHasUnresolvedRefs(entity) {
				// Fully resolved — remove from pending
				if err := a.PendingRepo.RemovePending(entity.EntityUUID); err != nil {
					return resolved, err
				}
				resolved++
			} else {
				// Still unresolved — increment retry
				_ = a.PendingRepo.IncrementPendingRetry(entity.EntityUUID)
			}
		default:
			// Non-task pending entities shouldn't happen, but clean them up
			_ = a.PendingRepo.RemovePending(entity.EntityUUID)
		}
	}
	return resolved, nil
}

// taskHasUnresolvedRefs checks if a task entity state has UUID references
// that couldn't be resolved to local IDs.
func (a *ApplyEntityStates) taskHasUnresolvedRefs(entity syncevent.EntityState) bool {
	eventType := syncevent.EventType(entity.EventType)
	if eventType == syncevent.EventTypeDeleted || entity.Snapshot == nil {
		return false
	}

	var snapshot syncevent.TaskSnapshotData
	if err := json.Unmarshal([]byte(*entity.Snapshot), &snapshot); err != nil {
		return false
	}

	if snapshot.ParentUUID != nil && a.TaskByUUIDLookup != nil {
		if _, err := a.TaskByUUIDLookup.GetByUUID(*snapshot.ParentUUID); err != nil {
			return true
		}
	}
	if snapshot.AreaUUID != nil && a.AreaByUUIDLookup != nil {
		if _, err := a.AreaByUUIDLookup.GetByUUID(*snapshot.AreaUUID); err != nil {
			return true
		}
	}
	if snapshot.RecurParentUUID != nil && a.TaskByUUIDLookup != nil {
		if _, err := a.TaskByUUIDLookup.GetByUUID(*snapshot.RecurParentUUID); err != nil {
			return true
		}
	}
	return false
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
	t := a.snapshotToTask(&snapshot)

	return a.TaskUpserter.Upsert(t)
}

// snapshotToTask converts a TaskSnapshotData to a Task, resolving UUIDs/names to local IDs.
func (a *ApplyEntityStates) snapshotToTask(snapshot *syncevent.TaskSnapshotData) *task.Task {
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
	}
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
