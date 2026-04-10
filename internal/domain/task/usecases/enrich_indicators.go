package usecases

import (
	"github.com/devbydaniel/tt/internal/domain/note"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// NoteChecker checks which entity UUIDs have notes.
type NoteChecker interface {
	HasNotesByUUIDs(et note.EntityType, uuids []string) (map[string]bool, error)
}

// EnrichIndicators populates HasNotes on a slice of tasks.
type EnrichIndicators struct {
	NoteChecker NoteChecker
}

func (e *EnrichIndicators) Execute(tasks []task.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	uuids := make([]string, len(tasks))
	for i := range tasks {
		uuids[i] = tasks[i].UUID
	}

	// Batch check notes — split by entity type
	if e.NoteChecker != nil {
		var taskUUIDs, projUUIDs []string
		taskIdx := make(map[string][]int)
		projIdx := make(map[string][]int)

		for i := range tasks {
			if tasks[i].IsProject() {
				projUUIDs = append(projUUIDs, tasks[i].UUID)
				projIdx[tasks[i].UUID] = append(projIdx[tasks[i].UUID], i)
			} else {
				taskUUIDs = append(taskUUIDs, tasks[i].UUID)
				taskIdx[tasks[i].UUID] = append(taskIdx[tasks[i].UUID], i)
			}
		}

		if len(taskUUIDs) > 0 {
			m, err := e.NoteChecker.HasNotesByUUIDs(note.EntityTask, taskUUIDs)
			if err != nil {
				return err
			}
			for uuid, has := range m {
				if has {
					for _, idx := range taskIdx[uuid] {
						tasks[idx].HasNotes = true
					}
				}
			}
		}

		if len(projUUIDs) > 0 {
			m, err := e.NoteChecker.HasNotesByUUIDs(note.EntityProject, projUUIDs)
			if err != nil {
				return err
			}
			for uuid, has := range m {
				if has {
					for _, idx := range projIdx[uuid] {
						tasks[idx].HasNotes = true
					}
				}
			}
		}
	}

	return nil
}
