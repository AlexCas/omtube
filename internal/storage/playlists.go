package storage

import (
	"database/sql"
	"fmt"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// Playlist es una lista de pistas con nombre único.
type Playlist struct {
	ID   int64
	Name string
}

// PlaylistRepo persiste playlists y su membresía de pistas (playlist_tracks),
// preservando el orden de inserción mediante la columna position.
type PlaylistRepo struct {
	db *sql.DB
}

// Create crea una playlist con el nombre dado y devuelve su id. El llamador es
// responsable de validar nombre vacío/duplicado; un nombre duplicado provoca un
// error por la restricción UNIQUE.
func (r *PlaylistRepo) Create(name string) (int64, error) {
	res, err := r.db.Exec(`INSERT INTO playlists (name) VALUES (?)`, name)
	if err != nil {
		return 0, fmt.Errorf("create playlist %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("create playlist %q: %w", name, err)
	}
	return id, nil
}

// Rename cambia el nombre de la playlist. Devuelve sql.ErrNoRows si el id no
// existe.
func (r *PlaylistRepo) Rename(id int64, name string) error {
	res, err := r.db.Exec(`UPDATE playlists SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("rename playlist %d: %w", id, err)
	}
	return requireRowsAffected(res, id)
}

// Delete elimina la playlist y su membresía (ON DELETE CASCADE). Devuelve
// sql.ErrNoRows si el id no existe.
func (r *PlaylistRepo) Delete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM playlists WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete playlist %d: %w", id, err)
	}
	return requireRowsAffected(res, id)
}

// List devuelve todas las playlists ordenadas por id (orden de creación).
func (r *PlaylistRepo) List() ([]Playlist, error) {
	rows, err := r.db.Query(`SELECT id, name FROM playlists ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	var out []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, fmt.Errorf("list playlists: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Add añade una pista al final de la playlist. Si la pista ya pertenece a la
// playlist, no hace nada (sin duplicado). La pista debe existir previamente en
// tracks.
func (r *PlaylistRepo) Add(playlistID int64, videoID string) error {
	const q = `
INSERT INTO playlist_tracks (playlist_id, video_id, position)
VALUES (
	?,
	?,
	(SELECT COALESCE(MAX(position), -1) + 1 FROM playlist_tracks WHERE playlist_id = ?)
)
ON CONFLICT(playlist_id, video_id) DO NOTHING`
	if _, err := r.db.Exec(q, playlistID, videoID, playlistID); err != nil {
		return fmt.Errorf("add track %q to playlist %d: %w", videoID, playlistID, err)
	}
	return nil
}

// Remove quita una pista de la playlist. Quitar una pista que no pertenece es
// un no-op sin error. El orden del resto se conserva (positions no se
// recompactan; el orden relativo se mantiene).
func (r *PlaylistRepo) Remove(playlistID int64, videoID string) error {
	const q = `DELETE FROM playlist_tracks WHERE playlist_id = ? AND video_id = ?`
	if _, err := r.db.Exec(q, playlistID, videoID); err != nil {
		return fmt.Errorf("remove track %q from playlist %d: %w", videoID, playlistID, err)
	}
	return nil
}

// Tracks devuelve las pistas de la playlist en orden de posición.
func (r *PlaylistRepo) Tracks(playlistID int64) ([]search.Result, error) {
	const q = `
SELECT t.video_id, t.title, t.uploader, t.duration
FROM playlist_tracks pt
JOIN tracks t ON t.video_id = pt.video_id
WHERE pt.playlist_id = ?
ORDER BY pt.position`
	rows, err := r.db.Query(q, playlistID)
	if err != nil {
		return nil, fmt.Errorf("tracks of playlist %d: %w", playlistID, err)
	}
	defer rows.Close()

	var out []search.Result
	for rows.Next() {
		var t search.Result
		if err := rows.Scan(&t.ID, &t.Title, &t.Uploader, &t.Duration); err != nil {
			return nil, fmt.Errorf("tracks of playlist %d: %w", playlistID, err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// requireRowsAffected devuelve sql.ErrNoRows si la sentencia no afectó filas
// (el id no existía).
func requireRowsAffected(res sql.Result, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("playlist %d: %w", id, sql.ErrNoRows)
	}
	return nil
}
