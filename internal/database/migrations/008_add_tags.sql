-- Task tags join table
CREATE TABLE task_tags (
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (task_id, tag_name)
);

-- Index for filtering by tag
CREATE INDEX idx_task_tags_tag_name ON task_tags(tag_name);
