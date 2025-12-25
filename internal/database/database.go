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

	// Enable foreign keys for CASCADE support
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	return &DB{Conn: conn}, nil
}

func (db *DB) Migrate() error {
	// Create migrations tracking table
	if _, err := db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return err
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()

		// Check if migration already applied
		var count int
		if err := db.Conn.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`,
			name,
		).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}

		if _, err := db.Conn.Exec(string(content)); err != nil {
			return err
		}

		// Record migration as applied
		if _, err := db.Conn.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))`,
			name,
		); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}
