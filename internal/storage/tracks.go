package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// TrackRepo persiste pistas identificadas por su video id (search.Result.ID).
type TrackRepo struct {
	db *sql.DB
}

// upsertTrackQuery inserta la pista o actualiza sus metadatos si ya existe
// (identidad por video_id). Los campos que pueden llegar vacíos desde una pista
// de origen incompleto (p. ej. una selección del historial, que no almacena
// duración) NO degradan el registro compartido: solo se sobreescriben cuando el
// valor entrante es no vacío, conservando en caso contrario el valor ya guardado.
const upsertTrackQuery = `
INSERT INTO tracks (video_id, title, uploader, duration)
VALUES (?, ?, ?, ?)
ON CONFLICT(video_id) DO UPDATE SET
	title    = CASE WHEN excluded.title    <> '' THEN excluded.title    ELSE tracks.title    END,
	uploader = CASE WHEN excluded.uploader <> '' THEN excluded.uploader ELSE tracks.uploader END,
	duration = CASE WHEN excluded.duration > 0   THEN excluded.duration ELSE tracks.duration END`

// Upsert inserta la pista o actualiza sus metadatos si ya existe (identidad por
// video_id). Garantiza que el registro exista antes de referenciarlo desde
// playlists, favoritos o historial.
func (r *TrackRepo) Upsert(t search.Result) error {
	if _, err := r.db.Exec(upsertTrackQuery, t.ID, t.Title, t.Uploader, t.Duration); err != nil {
		return fmt.Errorf("upsert track %q: %w", t.ID, err)
	}
	return nil
}

// UpsertTx es como Upsert pero ejecuta dentro de la transacción tx, para
// importaciones masivas atómicas (todo-o-nada).
func (r *TrackRepo) UpsertTx(tx *sql.Tx, t search.Result) error {
	if _, err := tx.Exec(upsertTrackQuery, t.ID, t.Title, t.Uploader, t.Duration); err != nil {
		return fmt.Errorf("upsert track %q: %w", t.ID, err)
	}
	return nil
}

// Get devuelve la pista por su video id. Si no existe, devuelve un Result
// vacío, found=false y err=nil (lectura de registro inexistente sin error).
func (r *TrackRepo) Get(id string) (search.Result, bool, error) {
	const q = `SELECT video_id, title, uploader, duration FROM tracks WHERE video_id = ?`
	var t search.Result
	err := r.db.QueryRow(q, id).Scan(&t.ID, &t.Title, &t.Uploader, &t.Duration)
	if errors.Is(err, sql.ErrNoRows) {
		return search.Result{}, false, nil
	}
	if err != nil {
		return search.Result{}, false, fmt.Errorf("get track %q: %w", id, err)
	}
	return t, true, nil
}
