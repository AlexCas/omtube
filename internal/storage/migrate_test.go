package storage

import (
	"path/filepath"
	"testing"
)

func tmpDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "library.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrateAdvancesUserVersionOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	v1, err := userVersion(db.SQL())
	if err != nil {
		t.Fatalf("userVersion: %v", err)
	}
	if want := len(migrations); v1 != want {
		t.Fatalf("user_version after first open = %d, want %d", v1, want)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reabrir la misma base: las migraciones ya aplicadas no deben re-ejecutarse
	// ni cambiar la versión (idempotente).
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer db2.Close()
	v2, err := userVersion(db2.SQL())
	if err != nil {
		t.Fatalf("userVersion: %v", err)
	}
	if v2 != v1 {
		t.Fatalf("user_version changed on reopen: %d -> %d", v1, v2)
	}
}

func TestMigrateIsIdempotentOnExistingTables(t *testing.T) {
	// Reabrir varias veces no debe fallar por "table already exists".
	path := filepath.Join(t.TempDir(), "library.db")
	for i := 0; i < 3; i++ {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("Open iteration %d: %v", i, err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close iteration %d: %v", i, err)
		}
	}
}
