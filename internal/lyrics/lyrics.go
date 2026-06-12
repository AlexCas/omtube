package lyrics

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// defaultBaseURL es el endpoint de lrclib, una API comunitaria sin auth que
// sirve letras sincronizadas (.lrc) y planas.
const defaultBaseURL = "https://lrclib.net"

// httpDoer abstrae *http.Client para poder inyectar un cliente de prueba.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Service resuelve letras para una pista: primero consulta la caché en base de
// datos y, ante un fallo, pide a lrclib. Cualquier error de red o ausencia de
// coincidencia se traduce en una Lyrics vacía sin propagar el error a la UI.
type Service struct {
	repo    *storage.LyricsRepo
	client  httpDoer
	baseURL string
}

// New construye el servicio de letras. repo es la caché persistente (puede ser
// nil para deshabilitar el cacheo); client es el cliente HTTP (nil ⇒ uno con
// timeout por defecto).
func New(repo *storage.LyricsRepo, client httpDoer) *Service {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Service{repo: repo, client: client, baseURL: defaultBaseURL}
}

// lrclibResponse es la forma de la respuesta de lrclib /api/get.
type lrclibResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

// Fetch resuelve la letra de la pista videoID con título/artista/duración. El
// orden es: caché en BD (acierto ⇒ sin HTTP) → lrclib. Prefiere letra
// sincronizada y cae a texto plano. Un fallo de API o un no-match devuelven una
// Lyrics vacía y err=nil: la reproducción nunca se bloquea por la letra.
func (s *Service) Fetch(ctx context.Context, videoID, title, artist string, dur int) (Lyrics, error) {
	if l, ok := s.fromCache(videoID); ok {
		return l, nil
	}

	body, synced, ok := s.fetchRemote(ctx, title, artist, dur)
	if !ok {
		return Lyrics{}, nil
	}

	var l Lyrics
	if synced {
		l = parseLRC(body)
	}
	if l.Empty() {
		l = plainText(body)
	}

	s.store(search.Result{ID: videoID, Title: title, Uploader: artist, Duration: dur}, body, l.Synced)
	return l, nil
}

// fromCache intenta resolver la letra desde la caché en BD. Devuelve ok=false
// cuando no hay repo, no hay fila o el cuerpo está vacío.
func (s *Service) fromCache(videoID string) (Lyrics, bool) {
	if s.repo == nil || videoID == "" {
		return Lyrics{}, false
	}
	entry, found, err := s.repo.Get(videoID)
	if err != nil || !found || entry.Body == "" {
		return Lyrics{}, false
	}
	if entry.Synced {
		if l := parseLRC(entry.Body); !l.Empty() {
			return l, true
		}
	}
	if l := plainText(entry.Body); !l.Empty() {
		return l, true
	}
	return Lyrics{}, false
}

// store guarda la letra resuelta en la caché en BD para evitar re-peticiones.
// Inserta la pista padre en tracks dentro de la misma transacción que la fila de
// letras, de modo que la FK lyrics_cache→tracks se satisfaga aunque ningún otro
// flujo haya insertado la pista todavía (una primera reproducción no convierte
// el cacheo en un no-op por la FK). Los errores se ignoran: el cacheo es una
// optimización, no debe romper el flujo de letras.
func (s *Service) store(track search.Result, body string, synced bool) {
	if s.repo == nil || track.ID == "" || body == "" {
		return
	}
	_ = s.repo.UpsertWithTrack(track, storage.LyricsEntry{VideoID: track.ID, Synced: synced, Body: body})
}

// fetchRemote consulta lrclib. Devuelve el cuerpo de letra (sincronizado si
// está disponible), el flag synced y ok=false ante cualquier error de red,
// código != 200 o ausencia de coincidencia.
func (s *Service) fetchRemote(ctx context.Context, title, artist string, dur int) (body string, synced, ok bool) {
	if title == "" {
		return "", false, false
	}

	q := url.Values{}
	q.Set("track_name", title)
	if artist != "" {
		q.Set("artist_name", artist)
	}
	if dur > 0 {
		q.Set("duration", strconv.Itoa(dur))
	}
	endpoint := s.baseURL + "/api/get?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, false
	}
	req.Header.Set("User-Agent", "terminaltube (https://github.com/alexcasdev/terminaltube)")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, false
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", false, false
	}

	var lr lrclibResponse
	if err := json.Unmarshal(data, &lr); err != nil {
		return "", false, false
	}
	if lr.SyncedLyrics != "" {
		return lr.SyncedLyrics, true, true
	}
	if lr.PlainLyrics != "" {
		return lr.PlainLyrics, false, true
	}
	return "", false, false
}
