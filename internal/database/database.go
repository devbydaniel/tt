package database

import (
	"database/sql"
	"embed"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	Conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DB{Conn: conn}, nil
}

func (db *DB) Migrate() error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}

		if _, err := db.Conn.Exec(string(content)); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}
