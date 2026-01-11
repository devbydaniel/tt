-- Migration 014: Add UUID column to areas for cross-device sync

-- Add uuid column (nullable initially for existing rows)
ALTER TABLE areas ADD COLUMN uuid TEXT;

-- Populate UUIDs for existing areas using SQLite's randomblob
-- Generates RFC 4122 version 4 UUIDs
UPDATE areas SET uuid = lower(hex(randomblob(4))) || '-' ||
                        lower(hex(randomblob(2))) || '-4' ||
                        substr(lower(hex(randomblob(2))), 2) || '-' ||
                        substr('89ab', abs(random()) % 4 + 1, 1) ||
                        substr(lower(hex(randomblob(2))), 2) || '-' ||
                        lower(hex(randomblob(6)))
WHERE uuid IS NULL;

-- Create unique index on uuid
CREATE UNIQUE INDEX idx_areas_uuid ON areas(uuid);
