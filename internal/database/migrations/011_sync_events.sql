-- Migration 011: Sync events for event stream synchronization

CREATE TABLE sync_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_uuid TEXT UNIQUE NOT NULL,
    entity_type TEXT NOT NULL,  -- 'task', 'area' (future)
    entity_uuid TEXT NOT NULL,
    client_id TEXT NOT NULL,
    event_type TEXT NOT NULL,   -- 'created', 'updated', 'deleted', 'completed'
    event_version INTEGER NOT NULL,
    timestamp TEXT NOT NULL,
    snapshot TEXT,              -- JSON blob, NULL for deletes

    -- Denormalized fields for efficient querying
    entity_title TEXT,
    entity_status TEXT
);

CREATE INDEX idx_sync_events_entity ON sync_events(entity_type, entity_uuid);
CREATE INDEX idx_sync_events_client_id ON sync_events(client_id);
CREATE INDEX idx_sync_events_timestamp ON sync_events(timestamp);
CREATE INDEX idx_sync_events_entity_version ON sync_events(entity_type, entity_uuid, event_version);
