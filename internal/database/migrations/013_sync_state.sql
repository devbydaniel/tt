-- Migration 013: Sync state for storing client sync cursor

CREATE TABLE sync_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
