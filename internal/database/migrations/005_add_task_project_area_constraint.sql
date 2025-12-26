-- Ensure a task cannot have both project_id and area_id set
CREATE TRIGGER task_project_area_exclusive_insert
BEFORE INSERT ON tasks
WHEN NEW.project_id IS NOT NULL AND NEW.area_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'task cannot have both project_id and area_id');
END;

CREATE TRIGGER task_project_area_exclusive_update
BEFORE UPDATE ON tasks
WHEN NEW.project_id IS NOT NULL AND NEW.area_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'task cannot have both project_id and area_id');
END;
