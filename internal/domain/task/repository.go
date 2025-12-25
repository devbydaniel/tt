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

func (r *Repository) Create(task *Task) error {
	result, err := r.db.Conn.Exec(
		`INSERT INTO tasks (uuid, title, status, created_at) VALUES (?, ?, ?, ?)`,
		task.UUID, task.Title, task.Status, task.CreatedAt.Format(time.RFC3339),
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

func (r *Repository) List() ([]Task, error) {
	rows, err := r.db.Conn.Query(
		`SELECT id, uuid, title, status, created_at, completed_at FROM tasks WHERE status = ? ORDER BY id`,
		StatusTodo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *Repository) GetByID(id int64) (*Task, error) {
	row := r.db.Conn.QueryRow(
		`SELECT id, uuid, title, status, created_at, completed_at FROM tasks WHERE id = ?`,
		id,
	)

	var t Task
	var createdAt string
	var completedAt *string
	if err := row.Scan(&t.ID, &t.UUID, &t.Title, &t.Status, &createdAt, &completedAt); err != nil {
		return nil, err
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

func (r *Repository) ListCompleted(since *time.Time) ([]Task, error) {
	var rows *sql.Rows
	var err error

	if since != nil {
		rows, err = r.db.Conn.Query(
			`SELECT id, uuid, title, status, created_at, completed_at
			 FROM tasks
			 WHERE status = ? AND completed_at >= ?
			 ORDER BY completed_at DESC`,
			StatusDone, since.Format(time.RFC3339),
		)
	} else {
		rows, err = r.db.Conn.Query(
			`SELECT id, uuid, title, status, created_at, completed_at
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

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		var t Task
		var createdAt string
		var completedAt *string
		if err := rows.Scan(&t.ID, &t.UUID, &t.Title, &t.Status, &createdAt, &completedAt); err != nil {
			return nil, err
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
