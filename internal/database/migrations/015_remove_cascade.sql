-- Migration 015: Remove CASCADE, use RESTRICT for foreign keys that need business logic
-- This forces explicit handling of cascaded operations in code (for sync events)

PRAGMA foreign_keys = OFF;

-- Step 1: Recreate tasks table with RESTRICT instead of CASCADE
CREATE TABLE tasks_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    task_type TEXT NOT NULL DEFAULT 'task',
    parent_id INTEGER REFERENCES tasks_new(id) ON DELETE RESTRICT,
    area_id INTEGER REFERENCES areas(id) ON DELETE RESTRICT,
    planned_date TEXT,
    due_date TEXT,
    state TEXT NOT NULL DEFAULT 'active',
    status TEXT NOT NULL DEFAULT 'todo',
    created_at TEXT NOT NULL,
    completed_at TEXT,
    recur_type TEXT,
    recur_rule TEXT,
    recur_end TEXT,
    recur_paused INTEGER NOT NULL DEFAULT 0,
    recur_parent_id INTEGER REFERENCES tasks_new(id)
);

-- Step 2: Copy data
INSERT INTO tasks_new SELECT * FROM tasks;

-- Step 3: Recreate task_tags (keep CASCADE - no business logic needed)
CREATE TABLE task_tags_new (
    task_id INTEGER NOT NULL REFERENCES tasks_new(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (task_id, tag_name)
);
INSERT INTO task_tags_new SELECT * FROM task_tags;

-- Step 4: Drop old tables
DROP TABLE task_tags;
DROP TABLE tasks;

-- Step 5: Rename new tables
ALTER TABLE tasks_new RENAME TO tasks;
ALTER TABLE task_tags_new RENAME TO task_tags;

-- Step 6: Recreate indexes
CREATE INDEX idx_task_tags_tag_name ON task_tags(tag_name);
CREATE UNIQUE INDEX idx_project_title_unique ON tasks(title) WHERE task_type = 'project';

-- Step 7: Recreate triggers
CREATE TRIGGER task_parent_area_exclusive_insert
BEFORE INSERT ON tasks
WHEN NEW.parent_id IS NOT NULL AND NEW.area_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'task cannot have both parent_id and area_id');
END;

CREATE TRIGGER task_parent_area_exclusive_update
BEFORE UPDATE ON tasks
WHEN NEW.parent_id IS NOT NULL AND NEW.area_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'task cannot have both parent_id and area_id');
END;

CREATE TRIGGER task_parent_must_be_project_insert
BEFORE INSERT ON tasks
WHEN NEW.parent_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'parent_id must reference a project')
    WHERE NOT EXISTS (
        SELECT 1 FROM tasks WHERE id = NEW.parent_id AND task_type = 'project'
    );
END;

CREATE TRIGGER task_parent_must_be_project_update
BEFORE UPDATE ON tasks
WHEN NEW.parent_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'parent_id must reference a project')
    WHERE NOT EXISTS (
        SELECT 1 FROM tasks WHERE id = NEW.parent_id AND task_type = 'project'
    );
END;

CREATE TRIGGER task_project_cannot_have_parent_insert
BEFORE INSERT ON tasks
WHEN NEW.task_type = 'project' AND NEW.parent_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'projects cannot have a parent');
END;

CREATE TRIGGER task_project_cannot_have_parent_update
BEFORE UPDATE ON tasks
WHEN NEW.task_type = 'project' AND NEW.parent_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'projects cannot have a parent');
END;

PRAGMA foreign_keys = ON;
