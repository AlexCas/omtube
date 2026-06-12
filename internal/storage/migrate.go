package storage

import (
	"database/sql"
	"fmt"
)

// migrations contiene las migraciones de esquema en orden. El índice 0 es la
// migración que lleva user_version de 0 a 1, el índice 1 de 1 a 2, etc. Añadir
// migraciones SOLO al final; nunca reordenar ni editar las existentes.
var migrations = []string{
	// Migración 1: esquema completo inicial.
	`
CREATE TABLE tracks (
	video_id TEXT PRIMARY KEY,
	title    TEXT NOT NULL,
	uploader TEXT NOT NULL,
	duration INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE playlists (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE playlist_tracks (
	playlist_id INTEGER NOT NULL,
	video_id    TEXT    NOT NULL,
	position    INTEGER NOT NULL,
	PRIMARY KEY (playlist_id, video_id),
	FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
	FOREIGN KEY (video_id)    REFERENCES tracks(video_id) ON DELETE CASCADE
);

CREATE TABLE favorites (
	video_id   TEXT PRIMARY KEY,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (video_id) REFERENCES tracks(video_id) ON DELETE CASCADE
);

CREATE TABLE history (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	video_id  TEXT NOT NULL,
	played_at TEXT NOT NULL,
	FOREIGN KEY (video_id) REFERENCES tracks(video_id) ON DELETE CASCADE
);
`,
	// Migración 2: índice de caché de audio y caché de letras. Solo AÑADE
	// tablas; los datos de la Fase 2 quedan intactos. cache_entries indexa los
	// archivos de audio descargados al directorio XDG cache; lyrics_cache guarda
	// las letras resueltas para evitar peticiones HTTP repetidas. Ambas tienen
	// FK a tracks ON DELETE CASCADE, de modo que borrar una pista limpia su caché.
	`
CREATE TABLE cache_entries (
	video_id   TEXT PRIMARY KEY,
	path       TEXT    NOT NULL,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	ext        TEXT    NOT NULL DEFAULT '',
	created_at TEXT    NOT NULL DEFAULT (datetime('now')),
	last_used  TEXT    NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (video_id) REFERENCES tracks(video_id) ON DELETE CASCADE
);

CREATE TABLE lyrics_cache (
	video_id   TEXT PRIMARY KEY,
	synced     INTEGER NOT NULL DEFAULT 0,
	body       TEXT    NOT NULL DEFAULT '',
	fetched_at TEXT    NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (video_id) REFERENCES tracks(video_id) ON DELETE CASCADE
);
`,
}

// migrate aplica todas las migraciones cuya versión es mayor que la
// user_version actual, en orden y cada una dentro de su propia transacción.
// Es idempotente: si la base ya está en la versión más reciente, no hace nada.
func migrate(db *sql.DB) error {
	current, err := userVersion(db)
	if err != nil {
		return err
	}

	for i := current; i < len(migrations); i++ {
		version := i + 1 // user_version objetivo tras aplicar migrations[i]
		if err := applyMigration(db, migrations[i], version); err != nil {
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
	}
	return nil
}

func userVersion(db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("read user_version: %w", err)
	}
	return v, nil
}

func applyMigration(db *sql.DB, stmt string, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(stmt); err != nil {
		_ = tx.Rollback()
		return err
	}
	// PRAGMA user_version no acepta parámetros enlazados; se interpola el
	// entero ya validado por el bucle de migrate.
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
