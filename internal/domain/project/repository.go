package project

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
)

var ErrProjectNotFound = errors.New("project not found")

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(project *Project) error {
	result, err := r.db.Conn.Exec(
		`INSERT INTO projects (name, area_id) VALUES (?, ?)`,
		project.Name, project.AreaID,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	project.ID = id
	return nil
}

func (r *Repository) List() ([]Project, error) {
	rows, err := r.db.Conn.Query(`SELECT id, name, area_id FROM projects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProjects(rows)
}

func (r *Repository) GetByID(id int64) (*Project, error) {
	row := r.db.Conn.QueryRow(`SELECT id, name, area_id FROM projects WHERE id = ?`, id)

	var p Project
	if err := row.Scan(&p.ID, &p.Name, &p.AreaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	return &p, nil
}

func (r *Repository) GetByName(name string) (*Project, error) {
	row := r.db.Conn.QueryRow(`SELECT id, name, area_id FROM projects WHERE name = ?`, name)

	var p Project
	if err := row.Scan(&p.ID, &p.Name, &p.AreaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	return &p, nil
}

func (r *Repository) Delete(id int64) error {
	result, err := r.db.Conn.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrProjectNotFound
	}

	return nil
}

func (r *Repository) ListByArea(areaID int64) ([]Project, error) {
	rows, err := r.db.Conn.Query(
		`SELECT id, name, area_id FROM projects WHERE area_id = ? ORDER BY name`,
		areaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProjects(rows)
}

func scanProjects(rows *sql.Rows) ([]Project, error) {
	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.AreaID); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, rows.Err()
}
