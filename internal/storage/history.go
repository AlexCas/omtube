package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// HistoryEntry es una pista reproducida con su marca de tiempo.
type HistoryEntry struct {
	Track    search.Result
	PlayedAt time.Time
}

// HistoryRepo persiste el historial de reproducción. La pista debe existir
// previamente en tracks.
type HistoryRepo struct {
	db *sql.DB
}

// insertHistoryQuery registra una pista como reproducida en el instante playedAt.
const insertHistoryQuery = `INSERT INTO history (video_id, played_at) VALUES (?, ?)`

// DB devuelve el *sql.DB subyacente para iniciar transacciones (p. ej. en
// importaciones masivas atómicas). Comparte la conexión con los demás repos.
func (r *HistoryRepo) DB() *sql.DB { return r.db }

// Insert registra una pista como reproducida en el instante playedAt.
func (r *HistoryRepo) Insert(videoID string, playedAt time.Time) error {
	if _, err := r.db.Exec(insertHistoryQuery, videoID, playedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert history %q: %w", videoID, err)
	}
	return nil
}

// InsertTx es como Insert pero ejecuta dentro de la transacción tx, para
// importaciones masivas atómicas (todo-o-nada).
func (r *HistoryRepo) InsertTx(tx *sql.Tx, videoID string, playedAt time.Time) error {
	if _, err := tx.Exec(insertHistoryQuery, videoID, playedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert history %q: %w", videoID, err)
	}
	return nil
}

// List devuelve las entradas de historial más recientes primero. Un limit <= 0
// devuelve todas las entradas.
func (r *HistoryRepo) List(limit int) ([]HistoryEntry, error) {
	q := `
SELECT t.video_id, t.title, t.uploader, t.duration, h.played_at
FROM history h
JOIN tracks t ON t.video_id = h.video_id
ORDER BY h.played_at DESC, h.id DESC`
	args := []any{}
	if limit > 0 {
		q += "\nLIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var out []HistoryEntry
	for rows.Next() {
		var (
			e  HistoryEntry
			ts string
		)
		if err := rows.Scan(&e.Track.ID, &e.Track.Title, &e.Track.Uploader, &e.Track.Duration, &ts); err != nil {
			return nil, fmt.Errorf("list history: %w", err)
		}
		t, perr := time.Parse(time.RFC3339Nano, ts)
		if perr != nil {
			return nil, fmt.Errorf("list history: parse played_at %q: %w", ts, perr)
		}
		e.PlayedAt = t
		out = append(out, e)
	}
	return out, rows.Err()
}
