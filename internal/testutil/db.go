package testutil

import (
	"testing"

	"github.com/devbydaniel/tt/internal/database"
)

// NewTestDB creates an in-memory SQLite database with all migrations applied.
// The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
