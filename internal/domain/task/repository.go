package task

import (
	"database/sql"
	"errors"
	"time"

	"github.com/devbydaniel/t/internal/database"
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
	var plannedDate, dueDate *string
	if task.PlannedDate != nil {
		s := task.PlannedDate.Format(dateFormat)
		plannedDate = &s
	}
	if task.DueDate != nil {
		s := task.DueDate.Format(dateFormat)
		dueDate = &s
	}

	result, err := r.db.Conn.Exec(
		`INSERT INTO tasks (uuid, title, project_id, area_id, planned_date, due_date, state, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.UUID, task.Title, task.ProjectID, task.AreaID, plannedDate, dueDate, task.State, task.Status, task.CreatedAt.Format(time.RFC3339),
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
	State     string // filter by state (active, someday)
	Today     bool   // planned_date = today OR overdue
	Upcoming  bool   // future planned/due dates
}

func (r *Repository) List(filter *ListFilter) ([]Task, error) {
	query := `SELECT id, uuid, title, project_id, area_id, planned_date, due_date, state, status, created_at, completed_at FROM tasks WHERE status = ?`
	args := []any{StatusTodo}

	if filter != nil {
		if filter.ProjectID != nil {
			query += ` AND project_id = ?`
			args = append(args, *filter.ProjectID)
		}
		if filter.AreaID != nil {
			query += ` AND area_id = ?`
			args = append(args, *filter.AreaID)
		}
		if filter.State != "" {
			query += ` AND state = ?`
			args = append(args, filter.State)
		}
		if filter.Today {
			// planned_date = today OR planned_date < today (overdue)
			today := time.Now().Format("2006-01-02")
			query += ` AND (date(planned_date) <= ? OR date(due_date) <= ?)`
			args = append(args, today, today)
		}
		if filter.Upcoming {
			// future planned_date or due_date
			today := time.Now().Format("2006-01-02")
			query += ` AND (date(planned_date) > ? OR date(due_date) > ?)`
			args = append(args, today, today)
		}
	}

	query += ` ORDER BY id`

	rows, err := r.db.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *Repository) GetByID(id int64) (*Task, error) {
	row := r.db.Conn.QueryRow(
		`SELECT id, uuid, title, project_id, area_id, planned_date, due_date, state, status, created_at, completed_at FROM tasks WHERE id = ?`,
		id,
	)

	var t Task
	var plannedDate, dueDate *string
	var createdAt string
	var completedAt *string
	if err := row.Scan(&t.ID, &t.UUID, &t.Title, &t.ProjectID, &t.AreaID, &plannedDate, &dueDate, &t.State, &t.Status, &createdAt, &completedAt); err != nil {
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
			`SELECT id, uuid, title, project_id, area_id, planned_date, due_date, state, status, created_at, completed_at
			 FROM tasks
			 WHERE status = ? AND completed_at >= ?
			 ORDER BY completed_at DESC`,
			StatusDone, since.Format(time.RFC3339),
		)
	} else {
		rows, err = r.db.Conn.Query(
			`SELECT id, uuid, title, project_id, area_id, planned_date, due_date, state, status, created_at, completed_at
			 FROM tasks
			 WHERE status = ?
			 ORDER BY completed_at DESC`,
			StatusDone,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *Repository) Update(task *Task) error {
	var plannedDate, dueDate *string
	if task.PlannedDate != nil {
		s := task.PlannedDate.Format(dateFormat)
		plannedDate = &s
	}
	if task.DueDate != nil {
		s := task.DueDate.Format(dateFormat)
		dueDate = &s
	}

	result, err := r.db.Conn.Exec(
		`UPDATE tasks SET title = ?, project_id = ?, area_id = ?, planned_date = ?, due_date = ?, state = ? WHERE id = ?`,
		task.Title, task.ProjectID, task.AreaID, plannedDate, dueDate, task.State, task.ID,
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
		if err := rows.Scan(&t.ID, &t.UUID, &t.Title, &t.ProjectID, &t.AreaID, &plannedDate, &dueDate, &t.State, &t.Status, &createdAt, &completedAt); err != nil {
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
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}
