package task

import (
	"time"

	"github.com/devbydaniel/t/internal/database"
)

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
		`SELECT id, uuid, title, status, created_at FROM tasks WHERE status = ? ORDER BY id`,
		StatusTodo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var createdAt string
		if err := rows.Scan(&t.ID, &t.UUID, &t.Title, &t.Status, &createdAt); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}
