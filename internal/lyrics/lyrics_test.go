package lyrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// newRepo abre una library.db temporal y registra la pista dada (la FK de
// lyrics_cache exige que exista), devolviendo el repositorio de letras.
func newRepo(t *testing.T, tracks ...search.Result) *storage.LyricsRepo {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, tr := range tracks {
		if err := db.Tracks().Upsert(tr); err != nil {
			t.Fatalf("Upsert track %q: %v", tr.ID, err)
		}
	}
	return db.Lyrics()
}

// newService construye un Service apuntado al servidor de prueba dado.
func newService(t *testing.T, repo *storage.LyricsRepo, srv *httptest.Server) *Service {
	t.Helper()
	s := New(repo, srv.Client())
	s.baseURL = srv.URL
	return s
}

func TestFetchSyncedFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"syncedLyrics":"[00:10.00]Hola\n[00:20.00]Mundo","plainLyrics":"Hola\nMundo"}`))
	}))
	defer srv.Close()

	s := newService(t, nil, srv)
	l, err := s.Fetch(context.Background(), "vid1", "Song", "Artist", 180)
	if err != nil {
		t.Fatalf("Fetch err inesperado: %v", err)
	}
	if !l.Synced || len(l.Lines) != 2 {
		t.Fatalf("se esperaba letra sincronizada de 2 líneas, got %+v", l)
	}
	if l.Lines[0].Text != "Hola" || l.Lines[1].T != 20 {
		t.Fatalf("contenido sincronizado incorrecto: %+v", l.Lines)
	}
}

func TestFetchPlainFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"syncedLyrics":"","plainLyrics":"Solo texto\nplano"}`))
	}))
	defer srv.Close()

	s := newService(t, nil, srv)
	l, err := s.Fetch(context.Background(), "vid2", "Song", "Artist", 0)
	if err != nil {
		t.Fatalf("Fetch err inesperado: %v", err)
	}
	if l.Synced || l.Plain != "Solo texto\nplano" {
		t.Fatalf("se esperaba texto plano, got %+v", l)
	}
}

func TestFetchNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":404,"message":"Not found"}`))
	}))
	defer srv.Close()

	s := newService(t, nil, srv)
	l, err := s.Fetch(context.Background(), "vid3", "Unknown", "Nobody", 0)
	if err != nil {
		t.Fatalf("Fetch no debe propagar error en no-match: %v", err)
	}
	if !l.Empty() {
		t.Fatalf("no-match debe dar Lyrics vacía, got %+v", l)
	}
}

func TestFetchAPIDown(t *testing.T) {
	// Servidor cerrado de inmediato ⇒ la conexión falla.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	s := New(nil, http.DefaultClient)
	s.baseURL = url
	l, err := s.Fetch(context.Background(), "vid4", "Song", "Artist", 0)
	if err != nil {
		t.Fatalf("Fetch no debe propagar error con API caída: %v", err)
	}
	if !l.Empty() {
		t.Fatalf("API caída debe dar Lyrics vacía, got %+v", l)
	}
}

func TestFetchCachesLyricsWithoutPreExistingTrack(t *testing.T) {
	// C1: con foreign_keys=ON y SIN pista sembrada, el cacheo de letras debe
	// insertar la pista padre y la fila de letras en la misma transacción, de modo
	// que la FK lyrics_cache→tracks no convierta el cacheo en un no-op silencioso.
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"syncedLyrics":"[00:01.00]Persistible","plainLyrics":""}`))
	}))
	defer srv.Close()

	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	track := search.Result{ID: "nopre", Title: "Song", Uploader: "Artist", Duration: 100}
	s := newService(t, db.Lyrics(), srv)

	// Primera Fetch: golpea HTTP y debe cachear pese a que la pista no existía.
	if _, err := s.Fetch(context.Background(), track.ID, track.Title, track.Uploader, track.Duration); err != nil {
		t.Fatalf("primera Fetch: %v", err)
	}

	// La pista padre quedó insertada (la FK se satisfizo en la misma transacción).
	if _, found, err := db.Tracks().Get("nopre"); err != nil || !found {
		t.Fatalf("Tracks().Get = (found=%v, err=%v), want (true, nil): no se insertó la pista padre", found, err)
	}
	// La fila de letras quedó persistida (no fue un no-op por FK).
	if _, found, err := db.Lyrics().Get("nopre"); err != nil || !found {
		t.Fatalf("Lyrics().Get = (found=%v, err=%v), want (true, nil): cacheo silenciosamente no-op", found, err)
	}

	// Segunda Fetch: debe resolverse desde la caché en BD, confirmando que se
	// persistió de verdad (sin segundo HTTP).
	if _, err := s.Fetch(context.Background(), track.ID, track.Title, track.Uploader, track.Duration); err != nil {
		t.Fatalf("segunda Fetch: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("la caché en BD debe evitar el segundo HTTP, hits=%d (cacheo no persistió)", got)
	}
}

func TestFetchCacheHitSkipsHTTP(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"syncedLyrics":"[00:01.00]Cacheable","plainLyrics":""}`))
	}))
	defer srv.Close()

	track := search.Result{ID: "vid5", Title: "Song", Uploader: "Artist", Duration: 100}
	repo := newRepo(t, track)
	s := newService(t, repo, srv)

	// Primera llamada: golpea HTTP y cachea en BD.
	if _, err := s.Fetch(context.Background(), track.ID, track.Title, track.Uploader, track.Duration); err != nil {
		t.Fatalf("primera Fetch: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("se esperaba 1 hit HTTP tras primera Fetch, got %d", got)
	}

	// Segunda llamada: debe resolverse desde la caché en BD sin HTTP.
	l, err := s.Fetch(context.Background(), track.ID, track.Title, track.Uploader, track.Duration)
	if err != nil {
		t.Fatalf("segunda Fetch: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("la caché en BD debe evitar el segundo HTTP, hits=%d", got)
	}
	if !l.Synced || len(l.Lines) != 1 || l.Lines[0].Text != "Cacheable" {
		t.Fatalf("letra cacheada incorrecta: %+v", l)
	}
}
