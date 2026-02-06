package syncevent

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devbydaniel/tt/internal/database"
)

var ErrEventNotFound = errors.New("sync event not found")

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new sync event into the database
func (r *Repository) Create(event *SyncEvent) error {
	result, err := r.db.Exec(
		`INSERT INTO sync_events (
			event_uuid, entity_type, entity_uuid, client_id,
			event_type, event_version, timestamp, snapshot,
			entity_title, entity_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventUUID,
		event.EntityType,
		event.EntityUUID,
		event.ClientID,
		event.EventType,
		event.EventVersion,
		event.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		event.Snapshot,
		event.EntityTitle,
		event.EntityStatus,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	event.ID = id
	return nil
}

// GetNextEventVersion returns the next version number for an entity.
// Returns 1 if no events exist for the entity.
func (r *Repository) GetNextEventVersion(entityType EntityType, entityUUID string) (int64, error) {
	var maxVersion sql.NullInt64
	err := r.db.QueryRow(
		`SELECT MAX(event_version) FROM sync_events
		 WHERE entity_type = ? AND entity_uuid = ?`,
		entityType, entityUUID,
	).Scan(&maxVersion)

	if err != nil {
		return 0, err
	}

	if !maxVersion.Valid {
		return 1, nil
	}

	return maxVersion.Int64 + 1, nil
}

// MaxFailureCount is the number of rejection cycles before an event is marked permanently failed.
const MaxFailureCount = 3

// GetUnpushed returns events that haven't been pushed to the sync server yet,
// excluding permanently failed events.
func (r *Repository) GetUnpushed(limit int) ([]*SyncEvent, error) {
	rows, err := r.db.Query(
		`SELECT id, event_uuid, entity_type, entity_uuid, client_id,
		        event_type, event_version, timestamp, snapshot,
		        entity_title, entity_status
		 FROM sync_events
		 WHERE pushed_at IS NULL AND permanently_failed = 0
		 ORDER BY timestamp ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*SyncEvent
	for rows.Next() {
		event, err := r.scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// MarkAsPushed updates the pushed_at timestamp for the given event UUIDs.
func (r *Repository) MarkAsPushed(eventUUIDs []string, pushedAt time.Time) error {
	if len(eventUUIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(eventUUIDs))
	args := make([]interface{}, len(eventUUIDs)+1)
	args[0] = pushedAt.Format("2006-01-02T15:04:05.000Z07:00")

	for i, uuid := range eventUUIDs {
		placeholders[i] = "?"
		args[i+1] = uuid
	}

	query := fmt.Sprintf(
		`UPDATE sync_events SET pushed_at = ? WHERE event_uuid IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	_, err := r.db.Exec(query, args...)
	return err
}

// GetByUUID retrieves a sync event by its UUID.
func (r *Repository) GetByUUID(eventUUID string) (*SyncEvent, error) {
	row := r.db.QueryRow(
		`SELECT id, event_uuid, entity_type, entity_uuid, client_id,
		        event_type, event_version, timestamp, snapshot,
		        entity_title, entity_status
		 FROM sync_events
		 WHERE event_uuid = ?`,
		eventUUID,
	)

	event, err := r.scanEventRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	return event, err
}

// scanEvent scans a row from sql.Rows into a SyncEvent.
func (r *Repository) scanEvent(rows *sql.Rows) (*SyncEvent, error) {
	var event SyncEvent
	var timestampStr string

	err := rows.Scan(
		&event.ID,
		&event.EventUUID,
		&event.EntityType,
		&event.EntityUUID,
		&event.ClientID,
		&event.EventType,
		&event.EventVersion,
		&timestampStr,
		&event.Snapshot,
		&event.EntityTitle,
		&event.EntityStatus,
	)
	if err != nil {
		return nil, err
	}

	event.Timestamp, err = time.Parse("2006-01-02T15:04:05.000Z07:00", timestampStr)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// scanEventRow scans a single row into a SyncEvent.
func (r *Repository) scanEventRow(row *sql.Row) (*SyncEvent, error) {
	var event SyncEvent
	var timestampStr string

	err := row.Scan(
		&event.ID,
		&event.EventUUID,
		&event.EntityType,
		&event.EntityUUID,
		&event.ClientID,
		&event.EventType,
		&event.EventVersion,
		&timestampStr,
		&event.Snapshot,
		&event.EntityTitle,
		&event.EntityStatus,
	)
	if err != nil {
		return nil, err
	}

	event.Timestamp, err = time.Parse("2006-01-02T15:04:05.000Z07:00", timestampStr)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// IncrementFailureCount increments the failure count for the given event UUIDs.
// Events that reach MaxFailureCount are automatically marked as permanently failed.
func (r *Repository) IncrementFailureCount(eventUUIDs []string) error {
	if len(eventUUIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(eventUUIDs))
	args := make([]interface{}, len(eventUUIDs))
	for i, uuid := range eventUUIDs {
		placeholders[i] = "?"
		args[i] = uuid
	}

	inClause := strings.Join(placeholders, ", ")

	// Increment failure count
	query := fmt.Sprintf(
		`UPDATE sync_events SET failure_count = failure_count + 1 WHERE event_uuid IN (%s)`,
		inClause,
	)
	if _, err := r.db.Exec(query, args...); err != nil {
		return err
	}

	// Mark as permanently failed if threshold reached
	query = fmt.Sprintf(
		`UPDATE sync_events SET permanently_failed = 1 WHERE failure_count >= %d AND event_uuid IN (%s)`,
		MaxFailureCount, inClause,
	)
	_, err := r.db.Exec(query, args...)
	return err
}

// GetPermanentlyFailed returns all permanently failed sync events.
func (r *Repository) GetPermanentlyFailed() ([]*SyncEvent, error) {
	rows, err := r.db.Query(
		`SELECT id, event_uuid, entity_type, entity_uuid, client_id,
		        event_type, event_version, timestamp, snapshot,
		        entity_title, entity_status
		 FROM sync_events
		 WHERE permanently_failed = 1
		 ORDER BY timestamp ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*SyncEvent
	for rows.Next() {
		event, err := r.scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// DeletePermanentlyFailed removes all permanently failed sync events.
// Returns the number of deleted events.
func (r *Repository) DeletePermanentlyFailed() (int64, error) {
	result, err := r.db.Exec("DELETE FROM sync_events WHERE permanently_failed = 1")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteAll removes all sync events from the database.
// Returns the number of deleted events.
func (r *Repository) DeleteAll() (int64, error) {
	result, err := r.db.Exec("DELETE FROM sync_events")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetMaxID returns the current maximum event ID (for cursor tracking).
// Returns 0 if no events exist.
func (r *Repository) GetMaxID() (int64, error) {
	var maxID sql.NullInt64
	err := r.db.QueryRow("SELECT MAX(id) FROM sync_events").Scan(&maxID)
	if err != nil {
		return 0, err
	}
	if !maxID.Valid {
		return 0, nil
	}
	return maxID.Int64, nil
}

// EntityState represents the latest state for a single entity.
type EntityState struct {
	EntityType string  `json:"entityType"`
	EntityUUID string  `json:"entityUuid"`
	EventType  string  `json:"eventType"` // "created", "updated", "deleted", "completed"
	Snapshot   *string `json:"snapshot"`  // JSON, nil for deletes
}

// GetLatestStatesSince returns the latest snapshot per entity since cursor,
// excluding entities whose latest event is in excludeEventUUIDs (the ones client just pushed).
// If another client made a newer change to the same entity, it will still be returned.
// Returns the entity states and the new cursor (max ID seen).
func (r *Repository) GetLatestStatesSince(cursor int64, excludeEventUUIDs []string) ([]EntityState, int64, error) {
	// Build the exclusion clause - exclude entities only if their latest event
	// is one the client just pushed, not if any event matches
	var excludeClause string
	var args []any
	args = append(args, cursor)

	if len(excludeEventUUIDs) > 0 {
		placeholders := make([]string, len(excludeEventUUIDs))
		for i, uuid := range excludeEventUUIDs {
			placeholders[i] = "?"
			args = append(args, uuid)
		}
		excludeClause = fmt.Sprintf(" AND e.event_uuid NOT IN (%s)", strings.Join(placeholders, ", "))
	}

	// Query to get the latest event per entity since cursor.
	// The exclusion is applied AFTER finding the latest event per entity,
	// so we only skip entities where the latest event itself was pushed by this client.
	// If a third party made a newer change, their event will be the latest and won't be excluded.
	query := fmt.Sprintf(`
		SELECT e.entity_type, e.entity_uuid, e.event_type, e.snapshot
		FROM sync_events e
		INNER JOIN (
			SELECT entity_uuid, MAX(id) as max_id
			FROM sync_events
			WHERE id > ?
			GROUP BY entity_uuid
		) latest ON e.entity_uuid = latest.entity_uuid AND e.id = latest.max_id
		WHERE 1=1%s
		ORDER BY e.id ASC
	`, excludeClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var states []EntityState
	for rows.Next() {
		var state EntityState
		if err := rows.Scan(&state.EntityType, &state.EntityUUID, &state.EventType, &state.Snapshot); err != nil {
			return nil, 0, err
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Get the new cursor (max ID)
	newCursor, err := r.GetMaxID()
	if err != nil {
		return nil, 0, err
	}

	return states, newCursor, nil
}

// GetSyncState retrieves a sync state value by key.
// Returns empty string if key doesn't exist.
func (r *Repository) GetSyncState(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM sync_state WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetSyncState stores a sync state value.
func (r *Repository) SetSyncState(key, value string) error {
	_, err := r.db.Exec(
		"INSERT INTO sync_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

// MaxPendingRetries is the maximum number of times a pending entity will be retried
// before being considered permanently unresolvable.
const MaxPendingRetries = 10

// SavePending inserts or updates an entity state in the pending resolution queue.
func (r *Repository) SavePending(state EntityState) error {
	_, err := r.db.Exec(
		`INSERT INTO pending_sync_resolution (entity_type, entity_uuid, event_type, snapshot, retry_count, updated_at)
		 VALUES (?, ?, ?, ?, 0, datetime('now'))
		 ON CONFLICT(entity_uuid) DO UPDATE SET
		   event_type = excluded.event_type,
		   snapshot = excluded.snapshot,
		   updated_at = datetime('now')`,
		state.EntityType, state.EntityUUID, state.EventType, state.Snapshot,
	)
	return err
}

// GetPending returns all pending entity states that haven't exceeded the retry limit.
func (r *Repository) GetPending() ([]EntityState, error) {
	rows, err := r.db.Query(
		`SELECT entity_type, entity_uuid, event_type, snapshot
		 FROM pending_sync_resolution
		 WHERE retry_count < ?
		 ORDER BY id ASC`,
		MaxPendingRetries,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var states []EntityState
	for rows.Next() {
		var s EntityState
		if err := rows.Scan(&s.EntityType, &s.EntityUUID, &s.EventType, &s.Snapshot); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

// IncrementPendingRetry increments the retry count for a pending entity.
func (r *Repository) IncrementPendingRetry(entityUUID string) error {
	_, err := r.db.Exec(
		`UPDATE pending_sync_resolution SET retry_count = retry_count + 1, updated_at = datetime('now')
		 WHERE entity_uuid = ?`,
		entityUUID,
	)
	return err
}

// RemovePending removes a resolved entity from the pending queue.
func (r *Repository) RemovePending(entityUUID string) error {
	_, err := r.db.Exec(
		`DELETE FROM pending_sync_resolution WHERE entity_uuid = ?`,
		entityUUID,
	)
	return err
}

// DeleteAllPending removes all entries from the pending resolution queue.
func (r *Repository) DeleteAllPending() error {
	_, err := r.db.Exec("DELETE FROM pending_sync_resolution")
	return err
}
