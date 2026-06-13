package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// LyricsEntry es una fila de la caché de letras: el cuerpo crudo (.lrc o texto
// plano) resuelto para una pista, con el flag que indica si tiene marcas de
// tiempo (sincronizado).
type LyricsEntry struct {
	VideoID   string
	Synced    bool
	Body      string
	FetchedAt string
	// Query es la consulta con la que el usuario encontró la letra manualmente.
	Query string
	// ProviderID es la referencia del proveedor (id de pista de lrclib) con la
	// que se resolvió la letra, para reusarla en re-reproducciones.
	ProviderID string
}

// LyricsRepo persiste las letras resueltas para evitar peticiones HTTP
// repetidas. La pista debe existir previamente en tracks (FK ON DELETE CASCADE).
type LyricsRepo struct {
	db *sql.DB
}

// upsertLyricsQuery registra o actualiza las letras cacheadas de una pista.
// Renueva fetched_at a "ahora" en cada llamada. query y provider_id solo se
// sobrescriben cuando el valor entrante no está vacío: así un re-fetch automático
// (que no aporta referencia) no borra la referencia que el usuario guardó con una
// búsqueda manual.
const upsertLyricsQuery = `
INSERT INTO lyrics_cache (video_id, synced, body, query, provider_id)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(video_id) DO UPDATE SET
	synced      = excluded.synced,
	body        = excluded.body,
	fetched_at  = datetime('now'),
	query       = CASE WHEN excluded.query       <> '' THEN excluded.query       ELSE lyrics_cache.query       END,
	provider_id = CASE WHEN excluded.provider_id <> '' THEN excluded.provider_id ELSE lyrics_cache.provider_id END`

func syncedFlag(synced bool) int {
	if synced {
		return 1
	}
	return 0
}

// Upsert registra o actualiza las letras cacheadas de una pista. La pista debe
// existir previamente en tracks (FK); para garantizarlo en una sola operación
// atómica usa UpsertWithTrack.
func (r *LyricsRepo) Upsert(e LyricsEntry) error {
	if _, err := r.db.Exec(upsertLyricsQuery, e.VideoID, syncedFlag(e.Synced), e.Body, e.Query, e.ProviderID); err != nil {
		return fmt.Errorf("upsert lyrics %q: %w", e.VideoID, err)
	}
	return nil
}

// UpsertWithTrack inserta primero la pista padre en tracks y luego las letras,
// ambas dentro de la misma transacción. Garantiza que la FK lyrics_cache→tracks
// se satisfaga sin depender de un flujo concurrente: el cacheo de letras de una
// primera reproducción nunca se convierte en un no-op silencioso por la FK.
func (r *LyricsRepo) UpsertWithTrack(track search.Result, e LyricsEntry) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("upsert lyrics %q: begin tx: %w", e.VideoID, err)
	}
	if _, err := tx.Exec(upsertTrackQuery, track.ID, track.Title, track.Uploader, track.Duration); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("upsert lyrics %q: upsert track: %w", e.VideoID, err)
	}
	if _, err := tx.Exec(upsertLyricsQuery, e.VideoID, syncedFlag(e.Synced), e.Body, e.Query, e.ProviderID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("upsert lyrics %q: %w", e.VideoID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("upsert lyrics %q: commit: %w", e.VideoID, err)
	}
	return nil
}

// Get devuelve las letras cacheadas por video id. Si no existen, devuelve una
// entrada vacía, found=false y err=nil.
func (r *LyricsRepo) Get(videoID string) (LyricsEntry, bool, error) {
	const q = `SELECT video_id, synced, body, fetched_at, query, provider_id FROM lyrics_cache WHERE video_id = ?`
	var (
		e      LyricsEntry
		synced int
	)
	err := r.db.QueryRow(q, videoID).Scan(&e.VideoID, &synced, &e.Body, &e.FetchedAt, &e.Query, &e.ProviderID)
	if errors.Is(err, sql.ErrNoRows) {
		return LyricsEntry{}, false, nil
	}
	if err != nil {
		return LyricsEntry{}, false, fmt.Errorf("get lyrics %q: %w", videoID, err)
	}
	e.Synced = synced != 0
	return e, true, nil
}
