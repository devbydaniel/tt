package syncevent

import "time"

// EntityType represents the type of entity being synced
type EntityType string

const (
	EntityTypeTask EntityType = "task"
	EntityTypeArea EntityType = "area" // Future
)

// EventType represents the type of sync operation
type EventType string

const (
	EventTypeCreated   EventType = "created"
	EventTypeUpdated   EventType = "updated"
	EventTypeDeleted   EventType = "deleted"
	EventTypeCompleted EventType = "completed"
)

// SyncEvent represents a single sync event with a full entity snapshot
type SyncEvent struct {
	ID           int64      `json:"id"`
	EventUUID    string     `json:"eventUuid"`
	EntityType   EntityType `json:"entityType"`
	EntityUUID   string     `json:"entityUuid"`
	ClientID     string     `json:"clientId"`
	EventType    EventType  `json:"eventType"`
	EventVersion int64      `json:"eventVersion"`
	Timestamp    time.Time  `json:"timestamp"`
	Snapshot     *string    `json:"snapshot"`
	EntityTitle  *string    `json:"entityTitle,omitempty"`
	EntityStatus *string    `json:"entityStatus,omitempty"`
	PushedAt     *time.Time `json:"pushedAt,omitempty"` // When event was pushed to sync server (nil = not pushed)
}
