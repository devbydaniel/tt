-- Add pushed_at column to track which events have been synced to the server
ALTER TABLE sync_events ADD COLUMN pushed_at TEXT;

-- Index for querying unpushed events
CREATE INDEX idx_sync_events_pushed_at ON sync_events(pushed_at);
