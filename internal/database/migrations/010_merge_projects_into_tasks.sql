-- Migration 010: Merge projects into tasks
-- Projects become tasks with task_type='project'
-- project_id is renamed to parent_id

-- Disable foreign keys during migration (they'll be re-enabled after)
PRAGMA foreign_keys = OFF;

-- Step 1: Create new tasks table with updated schema
CREATE TABLE tasks_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    task_type TEXT NOT NULL DEFAULT 'task',
    parent_id INTEGER REFERENCES tasks_new(id) ON DELETE CASCADE,
    area_id INTEGER REFERENCES areas(id) ON DELETE CASCADE,
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

-- Step 2: Copy existing tasks to new table (task_type='task', project_id becomes parent_id temporarily as old project ID)
INSERT INTO tasks_new (id, uuid, title, description, task_type, parent_id, area_id, planned_date, due_date, state, status, created_at, completed_at, recur_type, recur_rule, recur_end, recur_paused, recur_parent_id)
SELECT id, uuid, title, description, 'task', project_id, area_id, planned_date, due_date, state, status, created_at, completed_at, recur_type, recur_rule, recur_end, recur_paused, recur_parent_id
FROM tasks;

-- Step 3: Insert projects as tasks with task_type='project'
-- Generate UUIDs for projects using SQLite's hex(randomblob())
INSERT INTO tasks_new (uuid, title, task_type, area_id, state, status, created_at)
SELECT
    lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' ||
          substr(hex(randomblob(2)),2) || '-' ||
          substr('89ab', abs(random()) % 4 + 1, 1) ||
          substr(hex(randomblob(2)),2) || '-' ||
          hex(randomblob(6))) as uuid,
    name,
    'project',
    area_id,
    'active',
    'todo',
    datetime('now')
FROM projects;

-- Step 4: Create mapping table for old project IDs to new task IDs
CREATE TEMP TABLE project_mapping AS
SELECT p.id as old_project_id, t.id as new_task_id
FROM projects p
JOIN tasks_new t ON t.title = p.name AND t.task_type = 'project';

-- Step 5: Update parent_id in tasks_new to point to the new project task IDs
UPDATE tasks_new
SET parent_id = (
    SELECT new_task_id
    FROM project_mapping
    WHERE old_project_id = tasks_new.parent_id
)
WHERE parent_id IS NOT NULL AND task_type = 'task';

-- Step 6: Copy task_tags (no changes needed, just recreate for consistency)
CREATE TABLE task_tags_new (
    task_id INTEGER NOT NULL REFERENCES tasks_new(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (task_id, tag_name)
);
INSERT INTO task_tags_new SELECT * FROM task_tags;

-- Step 7: Drop old tables and triggers
DROP TRIGGER IF EXISTS task_project_area_exclusive_insert;
DROP TRIGGER IF EXISTS task_project_area_exclusive_update;
DROP TABLE task_tags;
DROP TABLE tasks;
DROP TABLE projects;

-- Step 8: Rename new tables
ALTER TABLE tasks_new RENAME TO tasks;
ALTER TABLE task_tags_new RENAME TO task_tags;

-- Step 9: Recreate indexes
CREATE INDEX idx_task_tags_tag_name ON task_tags(tag_name);

-- Step 10: Add constraint - task cannot have both parent_id AND area_id
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

-- Step 11: Add constraint - parent_id must reference a project-type task
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

-- Step 12: Add constraint - projects cannot have a parent (no nesting)
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

-- Step 13: Add partial unique index for project titles
CREATE UNIQUE INDEX idx_project_title_unique ON tasks(title) WHERE task_type = 'project';

-- Re-enable foreign keys
PRAGMA foreign_keys = ON;
