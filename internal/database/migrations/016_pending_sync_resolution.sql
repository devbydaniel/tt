-- Migration 016: Pending sync resolution queue for unresolved entity references
-- Stores entity states that couldn't fully resolve their references (parent, area, recur parent)
-- during apply. These are retried after each subsequent batch.

CREATE TABLE pending_sync_resolution (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_uuid TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    snapshot TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_pending_sync_entity ON pending_sync_resolution(entity_uuid);
