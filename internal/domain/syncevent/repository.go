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
	result, err := r.db.Conn.Exec(
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
	err := r.db.Conn.QueryRow(
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

// GetUnpushed returns events that haven't been pushed to the sync server yet.
func (r *Repository) GetUnpushed(limit int) ([]*SyncEvent, error) {
	rows, err := r.db.Conn.Query(
		`SELECT id, event_uuid, entity_type, entity_uuid, client_id,
		        event_type, event_version, timestamp, snapshot,
		        entity_title, entity_status
		 FROM sync_events
		 WHERE pushed_at IS NULL
		 ORDER BY timestamp ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

	_, err := r.db.Conn.Exec(query, args...)
	return err
}

// GetByUUID retrieves a sync event by its UUID.
func (r *Repository) GetByUUID(eventUUID string) (*SyncEvent, error) {
	row := r.db.Conn.QueryRow(
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
