-- Add recurrence support to tasks
ALTER TABLE tasks ADD COLUMN recur_type TEXT;
ALTER TABLE tasks ADD COLUMN recur_rule TEXT;
ALTER TABLE tasks ADD COLUMN recur_end TEXT;
ALTER TABLE tasks ADD COLUMN recur_paused INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN recur_parent_id INTEGER REFERENCES tasks(id);
