// Package storage implementa la capa de almacenamiento local en SQLite que
// respalda biblioteca, playlists, favoritos e historial. Usa el driver puro Go
// modernc.org/sqlite para conservar el binario único (sin cgo).
package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registra el driver "sqlite" (puro Go, sin cgo)
)

// DB envuelve la conexión a la base de datos local y expone los repositorios
// por entidad.
type DB struct {
	sql *sql.DB
}

// Open abre (o crea) la base de datos en path, aplica los pragmas
// recomendados (WAL y foreign_keys=ON) y ejecuta las migraciones pendientes.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite con database/sql: una sola conexión evita "database is locked"
	// en escrituras concurrentes y mantiene los pragmas por conexión activos.
	sqlDB.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("set pragma %q: %w", pragma, err)
		}
	}

	if err := migrate(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{sql: sqlDB}, nil
}

// Close cierra la conexión a la base de datos.
func (db *DB) Close() error { return db.sql.Close() }

// SQL devuelve el *sql.DB subyacente para construir repositorios.
func (db *DB) SQL() *sql.DB { return db.sql }

// Tracks devuelve el repositorio de pistas.
func (db *DB) Tracks() *TrackRepo { return &TrackRepo{db: db.sql} }

// Playlists devuelve el repositorio de playlists.
func (db *DB) Playlists() *PlaylistRepo { return &PlaylistRepo{db: db.sql} }

// Favorites devuelve el repositorio de favoritos.
func (db *DB) Favorites() *FavoriteRepo { return &FavoriteRepo{db: db.sql} }

// History devuelve el repositorio de historial.
func (db *DB) History() *HistoryRepo { return &HistoryRepo{db: db.sql} }

// Cache devuelve el repositorio del índice de caché de audio.
func (db *DB) Cache() *CacheRepo { return &CacheRepo{db: db.sql} }

// Lyrics devuelve el repositorio de caché de letras.
func (db *DB) Lyrics() *LyricsRepo { return &LyricsRepo{db: db.sql} }
