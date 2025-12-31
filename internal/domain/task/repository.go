package task

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/tt/internal/database"
)

var ErrTaskNotFound = errors.New("task not found")

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

const dateFormat = "2006-01-02"

func (r *Repository) Create(task *Task) error {
	var plannedDate, dueDate, recurEnd *string
	if task.PlannedDate != nil {
		s := task.PlannedDate.Format(dateFormat)
		plannedDate = &s
	}
	if task.DueDate != nil {
		s := task.DueDate.Format(dateFormat)
		dueDate = &s
	}
	if task.RecurEnd != nil {
		s := task.RecurEnd.Format(dateFormat)
		recurEnd = &s
	}

	result, err := r.db.Conn.Exec(
		`INSERT INTO tasks (uuid, title, description, project_id, area_id, planned_date, due_date, state, status, created_at, recur_type, recur_rule, recur_end, recur_paused, recur_parent_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.UUID, task.Title, task.Description, task.ProjectID, task.AreaID, plannedDate, dueDate, task.State, task.Status, task.CreatedAt.Format(time.RFC3339),
		task.RecurType, task.RecurRule, recurEnd, task.RecurPaused, task.RecurParentID,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	task.ID = id
	return nil
}

type ListFilter struct {
	ProjectID *int64
	AreaID    *int64
	State     State        // filter by state (active, someday)
	Today     bool         // planned_date = today OR overdue
	Upcoming  bool         // future planned/due dates
	Anytime   bool         // no planned_date and no due_date (active only)
	Inbox     bool         // no project, no area, no dates
	TagName   string       // filter by tag
	Search    string       // case-insensitive title search
	Sort      []SortOption // sort options (default: created desc)
}

// buildOrderByClause builds the ORDER BY clause from sort options
func buildOrderByClause(filter *ListFilter) string {
	sortOpts := DefaultSort()
	if filter != nil && len(filter.Sort) > 0 {
		sortOpts = filter.Sort
	}

	clause := " ORDER BY "
	for i, opt := range sortOpts {
		if i > 0 {
			clause += ", "
		}
		col := sortFieldToColumn(opt.Field)
		dir := "ASC"
		if opt.Direction == SortDesc {
			dir = "DESC"
		}
		// Handle NULLs: always put NULLs last for better task management UX
		// SQLite doesn't have NULLS FIRST/LAST, so we use a CASE expression
		if isNullableField(opt.Field) {
			// CASE WHEN col IS NULL THEN 1 ELSE 0 END puts NULLs last
			clause += "CASE WHEN " + col + " IS NULL THEN 1 ELSE 0 END, " + col + " " + dir
		} else {
			clause += col + " " + dir
		}
	}
	return clause
}

func sortFieldToColumn(f SortField) string {
	switch f {
	case SortByID:
		return "t.id"
	case SortByTitle:
		return "t.title"
	case SortByPlanned:
		return "t.planned_date"
	case SortByDue:
		return "t.due_date"
	case SortByCreated:
		return "t.created_at"
	case SortByProject:
		return "p.name"
	case SortByArea:
		return "COALESCE(a.name, pa.name)"
	default:
		return "t.id"
	}
}

func isNullableField(f SortField) bool {
	switch f {
	case SortByPlanned, SortByDue, SortByProject, SortByArea:
		return true
	default:
		return false
	}
}

func (r *Repository) List(filter *ListFilter) ([]Task, error) {
	query := `SELECT t.id, t.uuid, t.title, t.description, t.project_id, t.area_id, t.planned_date, t.due_date, t.state, t.status, t.created_at, t.completed_at, t.recur_type, t.recur_rule, t.recur_end, t.recur_paused, t.recur_parent_id, p.name, COALESCE(a.name, pa.name) FROM tasks t`
	query += ` LEFT JOIN projects p ON t.project_id = p.id`
	query += ` LEFT JOIN areas a ON t.area_id = a.id`
	query += ` LEFT JOIN areas pa ON p.area_id = pa.id`
	args := []any{}

	// Join with task_tags if filtering by tag
	if filter != nil && filter.TagName != "" {
		query += ` INNER JOIN task_tags tt ON t.id = tt.task_id`
	}

	query += ` WHERE t.status = ?`
	args = append(args, StatusTodo)

	if filter != nil {
		if filter.TagName != "" {
			query += ` AND tt.tag_name = ?`
			args = append(args, filter.TagName)
		}
		if filter.ProjectID != nil {
			query += ` AND t.project_id = ?`
			args = append(args, *filter.ProjectID)
		}
		if filter.AreaID != nil {
			query += ` AND t.area_id = ?`
			args = append(args, *filter.AreaID)
		}
		if filter.State != "" {
			query += ` AND t.state = ?`
			args = append(args, filter.State)
		}
		if filter.Today {
			// planned_date = today OR planned_date < today (overdue)
			today := time.Now().Format("2006-01-02")
			query += ` AND (date(t.planned_date) <= ? OR date(t.due_date) <= ?)`
			args = append(args, today, today)
		}
		if filter.Upcoming {
			// future planned_date or due_date
			today := time.Now().Format("2006-01-02")
			query += ` AND (date(t.planned_date) > ? OR date(t.due_date) > ?)`
			args = append(args, today, today)
		}
		if filter.Anytime {
			// no planned_date and no due_date, must have project or area (excludes inbox)
			// enforces active state (someday tasks are excluded)
			query += ` AND t.planned_date IS NULL AND t.due_date IS NULL AND (t.project_id IS NOT NULL OR t.area_id IS NOT NULL) AND t.state = ?`
			args = append(args, StateActive)
		}
		if filter.Inbox {
			// no project, no area, no planned_date, no due_date
			// enforces active state (someday tasks are excluded)
			query += ` AND t.project_id IS NULL AND t.area_id IS NULL AND t.planned_date IS NULL AND t.due_date IS NULL AND t.state = ?`
			args = append(args, StateActive)
		}
		if filter.Search != "" {
			query += ` AND t.title LIKE ? COLLATE NOCASE`
			args = append(args, "%"+filter.Search+"%")
		}
	}

	query += buildOrderByClause(filter)

	rows, err := r.db.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, err
	}

	// Load tags for all tasks
	if err := r.loadTagsForTasks(tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) GetByID(id int64) (*Task, error) {
	row := r.db.Conn.QueryRow(
		`SELECT id, uuid, title, description, project_id, area_id, planned_date, due_date, state, status, created_at, completed_at, recur_type, recur_rule, recur_end, recur_paused, recur_parent_id FROM tasks WHERE id = ?`,
		id,
	)

	var t Task
	var plannedDate, dueDate *string
	var createdAt string
	var completedAt *string
	var recurEnd *string
	if err := row.Scan(&t.ID, &t.UUID, &t.Title, &t.Description, &t.ProjectID, &t.AreaID, &plannedDate, &dueDate, &t.State, &t.Status, &createdAt, &completedAt, &t.RecurType, &t.RecurRule, &recurEnd, &t.RecurPaused, &t.RecurParentID); err != nil {
		return nil, err
	}
	if plannedDate != nil {
		parsed, _ := time.Parse(dateFormat, *plannedDate)
		t.PlannedDate = &parsed
	}
	if dueDate != nil {
		parsed, _ := time.Parse(dateFormat, *dueDate)
		t.DueDate = &parsed
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if completedAt != nil {
		parsed, _ := time.Parse(time.RFC3339, *completedAt)
		t.CompletedAt = &parsed
	}
	if recurEnd != nil {
		parsed, _ := time.Parse(dateFormat, *recurEnd)
		t.RecurEnd = &parsed
	}

	// Load tags
	tags, err := r.getTagsForTask(id)
	if err != nil {
		return nil, err
	}
	t.Tags = tags

	return &t, nil
}

func (r *Repository) Complete(id int64, completedAt time.Time) error {
	result, err := r.db.Conn.Exec(
		`UPDATE tasks SET status = ?, completed_at = ? WHERE id = ? AND status = ?`,
		StatusDone, completedAt.Format(time.RFC3339), id, StatusTodo,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (r *Repository) Uncomplete(id int64) error {
	result, err := r.db.Conn.Exec(
		`UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ? AND status = ?`,
		StatusTodo, id, StatusDone,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (r *Repository) Delete(id int64) error {
	result, err := r.db.Conn.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (r *Repository) ListCompleted(since *time.Time) ([]Task, error) {
	var rows *sql.Rows
	var err error

	if since != nil {
		rows, err = r.db.Conn.Query(
			`SELECT t.id, t.uuid, t.title, t.description, t.project_id, t.area_id, t.planned_date, t.due_date, t.state, t.status, t.created_at, t.completed_at, t.recur_type, t.recur_rule, t.recur_end, t.recur_paused, t.recur_parent_id, p.name, COALESCE(a.name, pa.name)
			 FROM tasks t
			 LEFT JOIN projects p ON t.project_id = p.id
			 LEFT JOIN areas a ON t.area_id = a.id
			 LEFT JOIN areas pa ON p.area_id = pa.id
			 WHERE t.status = ? AND t.completed_at >= ?
			 ORDER BY t.completed_at DESC`,
			StatusDone, since.Format(time.RFC3339),
		)
	} else {
		rows, err = r.db.Conn.Query(
			`SELECT t.id, t.uuid, t.title, t.description, t.project_id, t.area_id, t.planned_date, t.due_date, t.state, t.status, t.created_at, t.completed_at, t.recur_type, t.recur_rule, t.recur_end, t.recur_paused, t.recur_parent_id, p.name, COALESCE(a.name, pa.name)
			 FROM tasks t
			 LEFT JOIN projects p ON t.project_id = p.id
			 LEFT JOIN areas a ON t.area_id = a.id
			 LEFT JOIN areas pa ON p.area_id = pa.id
			 WHERE t.status = ?
			 ORDER BY t.completed_at DESC`,
			StatusDone,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, err
	}

	// Load tags for all tasks
	if err := r.loadTagsForTasks(tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) Update(task *Task) error {
	var plannedDate, dueDate, recurEnd *string
	if task.PlannedDate != nil {
		s := task.PlannedDate.Format(dateFormat)
		plannedDate = &s
	}
	if task.DueDate != nil {
		s := task.DueDate.Format(dateFormat)
		dueDate = &s
	}
	if task.RecurEnd != nil {
		s := task.RecurEnd.Format(dateFormat)
		recurEnd = &s
	}

	result, err := r.db.Conn.Exec(
		`UPDATE tasks SET title = ?, description = ?, project_id = ?, area_id = ?, planned_date = ?, due_date = ?, state = ?, recur_type = ?, recur_rule = ?, recur_end = ?, recur_paused = ? WHERE id = ?`,
		task.Title, task.Description, task.ProjectID, task.AreaID, plannedDate, dueDate, task.State, task.RecurType, task.RecurRule, recurEnd, task.RecurPaused, task.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		var t Task
		var plannedDate, dueDate *string
		var createdAt string
		var completedAt *string
		var recurEnd *string
		if err := rows.Scan(&t.ID, &t.UUID, &t.Title, &t.Description, &t.ProjectID, &t.AreaID, &plannedDate, &dueDate, &t.State, &t.Status, &createdAt, &completedAt, &t.RecurType, &t.RecurRule, &recurEnd, &t.RecurPaused, &t.RecurParentID, &t.ProjectName, &t.AreaName); err != nil {
			return nil, err
		}
		if plannedDate != nil {
			parsed, _ := time.Parse(dateFormat, *plannedDate)
			t.PlannedDate = &parsed
		}
		if dueDate != nil {
			parsed, _ := time.Parse(dateFormat, *dueDate)
			t.DueDate = &parsed
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if completedAt != nil {
			parsed, _ := time.Parse(time.RFC3339, *completedAt)
			t.CompletedAt = &parsed
		}
		if recurEnd != nil {
			parsed, _ := time.Parse(dateFormat, *recurEnd)
			t.RecurEnd = &parsed
		}
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// getTagsForTask returns all tag names for a single task
func (r *Repository) getTagsForTask(taskID int64) ([]string, error) {
	rows, err := r.db.Conn.Query(`SELECT tag_name FROM task_tags WHERE task_id = ? ORDER BY tag_name`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// loadTagsForTasks loads tags for multiple tasks efficiently
func (r *Repository) loadTagsForTasks(tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}

	// Build task ID list and index map
	ids := make([]any, len(tasks))
	idxMap := make(map[int64]int)
	for i := range tasks {
		ids[i] = tasks[i].ID
		idxMap[tasks[i].ID] = i
	}

	// Build placeholder string
	placeholders := "?"
	for i := 1; i < len(ids); i++ {
		placeholders += ",?"
	}

	rows, err := r.db.Conn.Query(
		`SELECT task_id, tag_name FROM task_tags WHERE task_id IN (`+placeholders+`) ORDER BY tag_name`,
		ids...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var taskID int64
		var tagName string
		if err := rows.Scan(&taskID, &tagName); err != nil {
			return err
		}
		if idx, ok := idxMap[taskID]; ok {
			tasks[idx].Tags = append(tasks[idx].Tags, tagName)
		}
	}
	return rows.Err()
}

// AddTag adds a tag to a task
func (r *Repository) AddTag(taskID int64, tagName string) error {
	_, err := r.db.Conn.Exec(
		`INSERT OR IGNORE INTO task_tags (task_id, tag_name) VALUES (?, ?)`,
		taskID, tagName,
	)
	return err
}

// RemoveTag removes a tag from a task
func (r *Repository) RemoveTag(taskID int64, tagName string) error {
	_, err := r.db.Conn.Exec(
		`DELETE FROM task_tags WHERE task_id = ? AND tag_name = ?`,
		taskID, tagName,
	)
	return err
}

// ListTags returns all unique tags in use
func (r *Repository) ListTags() ([]string, error) {
	rows, err := r.db.Conn.Query(`SELECT DISTINCT tag_name FROM task_tags ORDER BY tag_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// SetTags replaces all tags on a task
func (r *Repository) SetTags(taskID int64, tags []string) error {
	// Delete existing tags
	if _, err := r.db.Conn.Exec(`DELETE FROM task_tags WHERE task_id = ?`, taskID); err != nil {
		return err
	}

	// Insert new tags
	for _, tag := range tags {
		if _, err := r.db.Conn.Exec(
			`INSERT INTO task_tags (task_id, tag_name) VALUES (?, ?)`,
			taskID, tag,
		); err != nil {
			return err
		}
	}
	return nil
}
