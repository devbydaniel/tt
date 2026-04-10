package comment

import (
	"strings"
	"time"

	"github.com/devbydaniel/tt/internal/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(c *Comment) error {
	c.CreatedAt = time.Now()

	result, err := r.db.Exec(
		`INSERT INTO comments (task_id, author, body, created_at) VALUES (?, ?, ?, ?)`,
		c.TaskID, c.Author, c.Body, c.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	c.ID = id
	return nil
}

func (r *Repository) ListByTask(taskID int64) ([]Comment, error) {
	rows, err := r.db.Query(
		`SELECT id, task_id, author, body, created_at FROM comments WHERE task_id = ? ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var createdAt string
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

// HasCommentsByTaskIDs returns a set of task IDs that have at least one comment.
func (r *Repository) HasCommentsByTaskIDs(taskIDs []int64) (map[int64]bool, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(taskIDs))
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT DISTINCT task_id FROM comments WHERE task_id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]bool)
	for rows.Next() {
		var taskID int64
		if err := rows.Scan(&taskID); err != nil {
			return nil, err
		}
		result[taskID] = true
	}
	return result, rows.Err()
}

func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	return err
}
