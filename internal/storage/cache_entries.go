package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// CacheEntry es una fila del índice de caché de audio: asocia una pista con el
// archivo local descargado, su tamaño y las marcas de tiempo usadas para la
// expiración por antigüedad/uso.
type CacheEntry struct {
	VideoID   string
	Path      string
	SizeBytes int64
	Ext       string
	CreatedAt string
	LastUsed  string
}

// CacheRepo persiste el índice de archivos de audio cacheados. La pista debe
// existir previamente en tracks (FK ON DELETE CASCADE).
type CacheRepo struct {
	db *sql.DB
}

// upsertCacheEntryQuery registra o actualiza la entrada de caché de una pista.
// Renueva last_used a "ahora" en cada llamada para reflejar el acceso más
// reciente.
const upsertCacheEntryQuery = `
INSERT INTO cache_entries (video_id, path, size_bytes, ext)
VALUES (?, ?, ?, ?)
ON CONFLICT(video_id) DO UPDATE SET
	path       = excluded.path,
	size_bytes = excluded.size_bytes,
	ext        = excluded.ext,
	last_used  = datetime('now')`

// Upsert registra o actualiza la entrada de caché de una pista. La pista debe
// existir previamente en tracks (FK); para garantizarlo en una sola operación
// atómica usa UpsertWithTrack.
func (r *CacheRepo) Upsert(e CacheEntry) error {
	if _, err := r.db.Exec(upsertCacheEntryQuery, e.VideoID, e.Path, e.SizeBytes, e.Ext); err != nil {
		return fmt.Errorf("upsert cache entry %q: %w", e.VideoID, err)
	}
	return nil
}

// UpsertWithTrack inserta primero la pista padre en tracks y luego la entrada de
// caché, ambas dentro de la misma transacción. Garantiza que la FK
// cache_entries→tracks se satisfaga sin depender de que otro flujo concurrente
// (p. ej. el historial) haya insertado la pista antes: una primera reproducción
// nunca falla la FK ni deja el audio descargado sin indexar.
func (r *CacheRepo) UpsertWithTrack(track search.Result, e CacheEntry) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("upsert cache entry %q: begin tx: %w", e.VideoID, err)
	}
	if _, err := tx.Exec(upsertTrackQuery, track.ID, track.Title, track.Uploader, track.Duration); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("upsert cache entry %q: upsert track: %w", e.VideoID, err)
	}
	if _, err := tx.Exec(upsertCacheEntryQuery, e.VideoID, e.Path, e.SizeBytes, e.Ext); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("upsert cache entry %q: %w", e.VideoID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("upsert cache entry %q: commit: %w", e.VideoID, err)
	}
	return nil
}

// Get devuelve la entrada de caché por video id. Si no existe, devuelve una
// entrada vacía, found=false y err=nil.
func (r *CacheRepo) Get(videoID string) (CacheEntry, bool, error) {
	const q = `SELECT video_id, path, size_bytes, ext, created_at, last_used
FROM cache_entries WHERE video_id = ?`
	var e CacheEntry
	err := r.db.QueryRow(q, videoID).Scan(&e.VideoID, &e.Path, &e.SizeBytes, &e.Ext, &e.CreatedAt, &e.LastUsed)
	if errors.Is(err, sql.ErrNoRows) {
		return CacheEntry{}, false, nil
	}
	if err != nil {
		return CacheEntry{}, false, fmt.Errorf("get cache entry %q: %w", videoID, err)
	}
	return e, true, nil
}

// Touch renueva last_used a "ahora" para reflejar que la pista se reprodujo de
// nuevo. Tocar una entrada inexistente es un no-op sin error.
func (r *CacheRepo) Touch(videoID string) error {
	if _, err := r.db.Exec(`UPDATE cache_entries SET last_used = datetime('now') WHERE video_id = ?`, videoID); err != nil {
		return fmt.Errorf("touch cache entry %q: %w", videoID, err)
	}
	return nil
}

// Delete elimina la entrada de caché. Borrar una entrada inexistente es un
// no-op sin error.
func (r *CacheRepo) Delete(videoID string) error {
	if _, err := r.db.Exec(`DELETE FROM cache_entries WHERE video_id = ?`, videoID); err != nil {
		return fmt.Errorf("delete cache entry %q: %w", videoID, err)
	}
	return nil
}

// List devuelve todas las entradas de caché ordenadas por last_used ascendente
// (las menos usadas primero), para que la expiración elimine las más antiguas
// antes. Sin entradas devuelve una lista vacía sin error.
func (r *CacheRepo) List() ([]CacheEntry, error) {
	const q = `SELECT video_id, path, size_bytes, ext, created_at, last_used
FROM cache_entries ORDER BY last_used ASC, video_id`
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list cache entries: %w", err)
	}
	defer rows.Close()

	var out []CacheEntry
	for rows.Next() {
		var e CacheEntry
		if err := rows.Scan(&e.VideoID, &e.Path, &e.SizeBytes, &e.Ext, &e.CreatedAt, &e.LastUsed); err != nil {
			return nil, fmt.Errorf("list cache entries: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// TotalBytes devuelve la suma de size_bytes de todas las entradas de caché.
// Sin entradas devuelve 0 sin error.
func (r *CacheRepo) TotalBytes() (int64, error) {
	var total sql.NullInt64
	if err := r.db.QueryRow(`SELECT SUM(size_bytes) FROM cache_entries`).Scan(&total); err != nil {
		return 0, fmt.Errorf("total cache bytes: %w", err)
	}
	return total.Int64, nil
}
