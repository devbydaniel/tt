package database_test

import (
	"errors"
	"testing"

	"github.com/devbydaniel/tt/internal/testutil"
)

func TestRunInTxCommit(t *testing.T) {
	db := testutil.NewTestDB(t)

	err := db.RunInTx(func() error {
		_, err := db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-1', 'Test Area')`)
		return err
	})
	if err != nil {
		t.Fatalf("RunInTx() error = %v", err)
	}

	// Verify the row is visible after commit
	var name string
	err = db.QueryRow(`SELECT name FROM areas WHERE uuid = 'tx-1'`).Scan(&name)
	if err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if name != "Test Area" {
		t.Errorf("name = %q, want %q", name, "Test Area")
	}
}

func TestRunInTxRollback(t *testing.T) {
	db := testutil.NewTestDB(t)

	testErr := errors.New("something went wrong")
	err := db.RunInTx(func() error {
		_, err := db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-2', 'Should Not Exist')`)
		if err != nil {
			return err
		}
		return testErr
	})
	if !errors.Is(err, testErr) {
		t.Fatalf("RunInTx() error = %v, want %v", err, testErr)
	}

	// Verify the row was rolled back
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM areas WHERE uuid = 'tx-2'`).Scan(&count)
	if count != 0 {
		t.Errorf("row count = %d, want 0 (should be rolled back)", count)
	}
}

func TestRunInTxNested(t *testing.T) {
	db := testutil.NewTestDB(t)

	err := db.RunInTx(func() error {
		_, err := db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-outer', 'Outer')`)
		if err != nil {
			return err
		}

		// Nested RunInTx should be a no-op (uses outer transaction)
		return db.RunInTx(func() error {
			_, err := db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-inner', 'Inner')`)
			return err
		})
	})
	if err != nil {
		t.Fatalf("RunInTx() error = %v", err)
	}

	// Both rows should be committed
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM areas WHERE uuid IN ('tx-outer', 'tx-inner')`).Scan(&count)
	if count != 2 {
		t.Errorf("row count = %d, want 2", count)
	}
}

func TestRunInTxNestedRollback(t *testing.T) {
	db := testutil.NewTestDB(t)

	testErr := errors.New("inner failure")
	err := db.RunInTx(func() error {
		_, err := db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-nr-outer', 'Outer')`)
		if err != nil {
			return err
		}

		// Inner error propagates and causes outer rollback
		return db.RunInTx(func() error {
			_, err := db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-nr-inner', 'Inner')`)
			if err != nil {
				return err
			}
			return testErr
		})
	})
	if !errors.Is(err, testErr) {
		t.Fatalf("RunInTx() error = %v, want %v", err, testErr)
	}

	// Both rows should be rolled back
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM areas WHERE uuid IN ('tx-nr-outer', 'tx-nr-inner')`).Scan(&count)
	if count != 0 {
		t.Errorf("row count = %d, want 0 (both should be rolled back)", count)
	}
}

func TestRunInTxPanicRollback(t *testing.T) {
	db := testutil.NewTestDB(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic to propagate")
		}

		// Verify the row was rolled back
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM areas WHERE uuid = 'tx-panic'`).Scan(&count)
		if count != 0 {
			t.Errorf("row count = %d, want 0 (should be rolled back on panic)", count)
		}
	}()

	db.RunInTx(func() error {
		db.Exec(`INSERT INTO areas (uuid, name) VALUES ('tx-panic', 'Panic Area')`)
		panic("something terrible")
	})
}
