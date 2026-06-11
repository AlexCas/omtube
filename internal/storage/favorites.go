package storage

import (
	"database/sql"
	"fmt"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// FavoriteRepo persiste el estado de favorito de las pistas. La pista debe
// existir previamente en tracks.
type FavoriteRepo struct {
	db *sql.DB
}

// Add marca una pista como favorita. Es idempotente: marcar una pista ya
// favorita no crea un duplicado ni produce error.
func (r *FavoriteRepo) Add(videoID string) error {
	const q = `INSERT INTO favorites (video_id) VALUES (?) ON CONFLICT(video_id) DO NOTHING`
	if _, err := r.db.Exec(q, videoID); err != nil {
		return fmt.Errorf("add favorite %q: %w", videoID, err)
	}
	return nil
}

// Remove desmarca una pista como favorita. Desmarcar una pista que no es
// favorita es un no-op sin error.
func (r *FavoriteRepo) Remove(videoID string) error {
	if _, err := r.db.Exec(`DELETE FROM favorites WHERE video_id = ?`, videoID); err != nil {
		return fmt.Errorf("remove favorite %q: %w", videoID, err)
	}
	return nil
}

// Exists indica si la pista está marcada como favorita.
func (r *FavoriteRepo) Exists(videoID string) (bool, error) {
	const q = `SELECT 1 FROM favorites WHERE video_id = ?`
	var one int
	err := r.db.QueryRow(q, videoID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("favorite exists %q: %w", videoID, err)
	}
	return true, nil
}

// List devuelve las pistas favoritas, más recientes primero. Sin favoritos
// devuelve una lista vacía sin error.
func (r *FavoriteRepo) List() ([]search.Result, error) {
	const q = `
SELECT t.video_id, t.title, t.uploader, t.duration
FROM favorites f
JOIN tracks t ON t.video_id = f.video_id
ORDER BY f.created_at DESC, f.video_id`
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()

	var out []search.Result
	for rows.Next() {
		var t search.Result
		if err := rows.Scan(&t.ID, &t.Title, &t.Uploader, &t.Duration); err != nil {
			return nil, fmt.Errorf("list favorites: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
