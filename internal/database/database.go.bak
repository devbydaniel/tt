package database

import (
	"database/sql"
	"embed"

	_ "modernc.org/sqlite" // register sqlite driver
)

//go:embed migrations/*.sql
var migrations embed.FS

// DBTX is the common interface between *sql.DB and *sql.Tx.
type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type DB struct {
	Conn *sql.DB
	tx   *sql.Tx
}

// Exec delegates to the active transaction if present, otherwise to Conn.
func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	if db.tx != nil {
		return db.tx.Exec(query, args...)
	}
	return db.Conn.Exec(query, args...)
}

// Query delegates to the active transaction if present, otherwise to Conn.
func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	if db.tx != nil {
		return db.tx.Query(query, args...)
	}
	return db.Conn.Query(query, args...)
}

// QueryRow delegates to the active transaction if present, otherwise to Conn.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	if db.tx != nil {
		return db.tx.QueryRow(query, args...)
	}
	return db.Conn.QueryRow(query, args...)
}

// RunInTx executes fn within a database transaction. If a transaction is already
// active (nested call), fn is called directly without creating a new transaction.
// On error or panic, the transaction is rolled back.
func (db *DB) RunInTx(fn func() error) error {
	if db.tx != nil {
		// Nested call — just run fn directly
		return fn()
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}

	db.tx = tx
	defer func() {
		db.tx = nil
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-panic after rollback
		}
	}()

	if err := fn(); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
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
