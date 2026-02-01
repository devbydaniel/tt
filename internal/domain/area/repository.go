package area

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"

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
	if area.UUID == "" {
		area.UUID = uuid.New().String()
	}

	result, err := r.db.Conn.Exec(
		`INSERT INTO areas (uuid, name) VALUES (?, ?)`,
		area.UUID, area.Name,
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
	rows, err := r.db.Conn.Query(`SELECT id, uuid, name FROM areas ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanAreas(rows)
}

func (r *Repository) GetByID(id int64) (*Area, error) {
	row := r.db.Conn.QueryRow(`SELECT id, uuid, name FROM areas WHERE id = ?`, id)

	var a Area
	if err := row.Scan(&a.ID, &a.UUID, &a.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAreaNotFound
		}
		return nil, err
	}

	return &a, nil
}

func (r *Repository) GetByName(name string) (*Area, error) {
	row := r.db.Conn.QueryRow(`SELECT id, uuid, name FROM areas WHERE name = ?`, name)

	var a Area
	if err := row.Scan(&a.ID, &a.UUID, &a.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAreaNotFound
		}
		return nil, err
	}

	return &a, nil
}

func (r *Repository) GetByUUID(areaUUID string) (*Area, error) {
	row := r.db.Conn.QueryRow(`SELECT id, uuid, name FROM areas WHERE uuid = ?`, areaUUID)

	var a Area
	if err := row.Scan(&a.ID, &a.UUID, &a.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAreaNotFound
		}
		return nil, err
	}

	return &a, nil
}

// DeleteAll removes all areas from the database. Returns the number of deleted areas.
func (r *Repository) DeleteAll() (int64, error) {
	result, err := r.db.Conn.Exec("DELETE FROM areas")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
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

func (r *Repository) DeleteByUUID(areaUUID string) error {
	result, err := r.db.Conn.Exec(`DELETE FROM areas WHERE uuid = ?`, areaUUID)
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

// Upsert creates or updates an area by UUID.
func (r *Repository) Upsert(a *Area) error {
	existing, err := r.GetByUUID(a.UUID)
	if err != nil && !errors.Is(err, ErrAreaNotFound) {
		return err
	}

	if existing != nil {
		// Update existing area
		_, err = r.db.Conn.Exec(
			`UPDATE areas SET name = ? WHERE uuid = ?`,
			a.Name, a.UUID,
		)
		if err != nil {
			return err
		}
		a.ID = existing.ID
		return nil
	}

	// Create new area
	return r.Create(a)
}

func scanAreas(rows *sql.Rows) ([]Area, error) {
	var areas []Area
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.UUID, &a.Name); err != nil {
			return nil, err
		}
		areas = append(areas, a)
	}

	return areas, rows.Err()
}
