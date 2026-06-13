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

func TestMigrate2AddsCacheTables(t *testing.T) {
	db := tmpDB(t)

	// Las tablas de la migración 2 deben existir tras abrir.
	for _, tbl := range []string{"cache_entries", "lyrics_cache"} {
		var name string
		err := db.SQL().QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&name)
		if err != nil {
			t.Fatalf("table %q not found after migrations: %v", tbl, err)
		}
	}
}

func TestMigrate3AddsLyricsReferenceColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Sembrar una pista y una fila de letra ANTES de comprobar columnas: simula una
	// fila preexistente que debe conservar el default ('') en las columnas nuevas.
	if _, err := db.SQL().Exec(`INSERT INTO tracks (video_id, title, uploader) VALUES ('vid1','T','U')`); err != nil {
		t.Fatalf("seed track: %v", err)
	}
	if _, err := db.SQL().Exec(`INSERT INTO lyrics_cache (video_id, synced, body) VALUES ('vid1', 0, 'hola')`); err != nil {
		t.Fatalf("seed lyrics: %v", err)
	}

	var query, providerID string
	err = db.SQL().QueryRow(`SELECT query, provider_id FROM lyrics_cache WHERE video_id='vid1'`).
		Scan(&query, &providerID)
	if err != nil {
		t.Fatalf("select new columns: %v", err)
	}
	if query != "" || providerID != "" {
		t.Fatalf("columnas nuevas sin default vacío: query=%q provider_id=%q", query, providerID)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reabrir: idempotente, la migración 3 no se re-ejecuta (no falla por columna
	// duplicada) y la versión queda en la última.
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	v2, err := userVersion(db2.SQL())
	if err != nil {
		t.Fatalf("userVersion reopen: %v", err)
	}
	if v2 != len(migrations) {
		t.Fatalf("user_version after reopen = %d, want %d", v2, len(migrations))
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
