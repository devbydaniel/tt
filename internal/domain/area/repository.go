package area

import (
	"database/sql"
	"errors"

	"github.com/devbydaniel/tt/internal/database"
)

var ErrAreaNotFound = errors.New("area not found")

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(area *Area) error {
	result, err := r.db.Conn.Exec(
		`INSERT INTO areas (name) VALUES (?)`,
		area.Name,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	area.ID = id
	return nil
}

func (r *Repository) List() ([]Area, error) {
	rows, err := r.db.Conn.Query(`SELECT id, name FROM areas ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAreas(rows)
}

func (r *Repository) GetByID(id int64) (*Area, error) {
	row := r.db.Conn.QueryRow(`SELECT id, name FROM areas WHERE id = ?`, id)

	var a Area
	if err := row.Scan(&a.ID, &a.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAreaNotFound
		}
		return nil, err
	}

	return &a, nil
}

func (r *Repository) GetByName(name string) (*Area, error) {
	row := r.db.Conn.QueryRow(`SELECT id, name FROM areas WHERE name = ?`, name)

	var a Area
	if err := row.Scan(&a.ID, &a.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAreaNotFound
		}
		return nil, err
	}

	return &a, nil
}

func (r *Repository) Delete(id int64) error {
	result, err := r.db.Conn.Exec(`DELETE FROM areas WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAreaNotFound
	}

	return nil
}

func (r *Repository) Update(area *Area) error {
	result, err := r.db.Conn.Exec(
		`UPDATE areas SET name = ? WHERE id = ?`,
		area.Name, area.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAreaNotFound
	}

	return nil
}

func scanAreas(rows *sql.Rows) ([]Area, error) {
	var areas []Area
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			return nil, err
		}
		areas = append(areas, a)
	}

	return areas, rows.Err()
}
