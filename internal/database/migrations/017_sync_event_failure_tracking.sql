-- Migration 017: Track permanently failed sync events
-- Events that are repeatedly rejected by the server get marked as permanently failed
-- so they stop jamming the sync queue.

ALTER TABLE sync_events ADD COLUMN failure_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_events ADD COLUMN permanently_failed INTEGER NOT NULL DEFAULT 0;

CREATE INDEX idx_sync_events_permanently_failed ON sync_events(permanently_failed);
