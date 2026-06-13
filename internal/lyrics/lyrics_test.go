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
	l, err := s.Fetch(context.Background(), search.Result{ID: "vid1", Title: "Song", Uploader: "Artist", Duration: 180}, "Song", "Artist")
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
	l, err := s.Fetch(context.Background(), search.Result{ID: "vid2", Title: "Song", Uploader: "Artist"}, "Song", "Artist")
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
	l, err := s.Fetch(context.Background(), search.Result{ID: "vid3", Title: "Unknown", Uploader: "Nobody"}, "Unknown", "Nobody")
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
	l, err := s.Fetch(context.Background(), search.Result{ID: "vid4", Title: "Song", Uploader: "Artist"}, "Song", "Artist")
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
	if _, err := s.Fetch(context.Background(), track, track.Title, track.Uploader); err != nil {
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
	if _, err := s.Fetch(context.Background(), track, track.Title, track.Uploader); err != nil {
		t.Fatalf("segunda Fetch: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("la caché en BD debe evitar el segundo HTTP, hits=%d (cacheo no persistió)", got)
	}
}

func TestFetchPreservesRawTrackIdentityInStorage(t *testing.T) {
	// Regresión: el cacheo de letras debe persistir la identidad CRUDA de la
	// pista (el título/uploader originales de YouTube), NO las cadenas
	// normalizadas usadas para la consulta saliente. De lo contrario la fila
	// compartida en tracks (historial/favoritos/playlists) quedaría reescrita
	// con el texto recortado en cada primera reproducción.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"syncedLyrics":"[00:01.00]Letra","plainLyrics":""}`))
	}))
	defer srv.Close()

	// Pista cruda tal como la guardaría history.Add: título de YouTube completo.
	const rawTitle = "Artist - Song (Official Music Video) [HD]"
	const rawUploader = "Artist Official"
	raw := search.Result{ID: "vidraw", Title: rawTitle, Uploader: rawUploader, Duration: 180}

	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Tracks().Upsert(raw); err != nil {
		t.Fatalf("Upsert pista cruda: %v", err)
	}

	s := newService(t, db.Lyrics(), srv)

	// Cadenas de consulta DIVERGENTES (normalizadas) que sí resuelven letra.
	const queryTitle = "Song"
	const queryArtist = "Artist"
	if queryTitle == rawTitle || queryArtist == rawUploader {
		t.Fatal("la prueba exige cadenas de consulta divergentes de las crudas")
	}

	if _, err := s.Fetch(context.Background(), raw, queryTitle, queryArtist); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// La fila compartida en tracks debe conservar el título/uploader CRUDOS.
	got, found, err := db.Tracks().Get(raw.ID)
	if err != nil || !found {
		t.Fatalf("Tracks().Get = (found=%v, err=%v), want (true, nil)", found, err)
	}
	if got.Title != rawTitle {
		t.Errorf("tracks.title reescrito: got %q, want %q (no debe normalizarse)", got.Title, rawTitle)
	}
	if got.Uploader != rawUploader {
		t.Errorf("tracks.uploader reescrito: got %q, want %q (no debe normalizarse)", got.Uploader, rawUploader)
	}
}

func TestFetchSearchFallbackAfterGetMiss(t *testing.T) {
	var getHits, searchHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/get":
			atomic.AddInt32(&getHits, 1)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":404,"message":"Not found"}`))
		case "/api/search":
			atomic.AddInt32(&searchHits, 1)
			_, _ = w.Write([]byte(`[{"trackName":"Song","artistName":"Artist","duration":180,"syncedLyrics":"[00:05.00]Encontrada","plainLyrics":"Encontrada"}]`))
		default:
			t.Errorf("ruta inesperada: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newService(t, nil, srv)
	l, err := s.Fetch(context.Background(), search.Result{ID: "vidsf", Title: "Song", Uploader: "Artist", Duration: 180}, "Song", "Artist")
	if err != nil {
		t.Fatalf("Fetch err inesperado: %v", err)
	}
	if !l.Synced || len(l.Lines) != 1 || l.Lines[0].Text != "Encontrada" {
		t.Fatalf("se esperaba letra resuelta por /api/search, got %+v", l)
	}
	if atomic.LoadInt32(&getHits) != 1 {
		t.Fatalf("se esperaba 1 golpe a /api/get, got %d", getHits)
	}
	if atomic.LoadInt32(&searchHits) != 1 {
		t.Fatalf("se esperaba 1 golpe a /api/search tras el miss, got %d", searchHits)
	}
}

func TestFetchSearchFallbackDisabled(t *testing.T) {
	var searchHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search" {
			atomic.AddInt32(&searchHits, 1)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":404,"message":"Not found"}`))
	}))
	defer srv.Close()

	s := newService(t, nil, srv)
	s.SetSearchFallback(false)
	l, err := s.Fetch(context.Background(), search.Result{ID: "vidno", Title: "Song", Uploader: "Artist"}, "Song", "Artist")
	if err != nil {
		t.Fatalf("Fetch err inesperado: %v", err)
	}
	if !l.Empty() {
		t.Fatalf("con fallback desactivado el miss debe dar Lyrics vacía, got %+v", l)
	}
	if got := atomic.LoadInt32(&searchHits); got != 0 {
		t.Fatalf("con fallback desactivado no debe llamarse a /api/search, hits=%d", got)
	}
}

func TestSearchReturnsCandidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "numb linkin" {
			t.Errorf("q = %q, want \"numb linkin\"", got)
		}
		_, _ = w.Write([]byte(`[
			{"id":111,"trackName":"Numb","artistName":"Linkin Park","duration":185,"syncedLyrics":"[00:01.00]a","plainLyrics":"a"},
			{"id":222,"trackName":"Numb (live)","artistName":"Linkin Park","duration":190,"syncedLyrics":"","plainLyrics":"texto"},
			{"id":333,"trackName":"Sin letra","artistName":"X","duration":0,"syncedLyrics":"","plainLyrics":""}
		]`))
	}))
	defer srv.Close()

	s := newService(t, nil, srv)
	cands, err := s.Search(context.Background(), "numb linkin")
	if err != nil {
		t.Fatalf("Search err: %v", err)
	}
	// El candidato sin letra (id 333) se descarta.
	if len(cands) != 2 {
		t.Fatalf("se esperaban 2 candidatos con letra, got %d: %+v", len(cands), cands)
	}
	if cands[0].ProviderID != "111" || !cands[0].Synced || cands[0].Query != "numb linkin" {
		t.Fatalf("candidato 0 incorrecto: %+v", cands[0])
	}
	if cands[1].ProviderID != "222" || cands[1].Synced {
		t.Fatalf("candidato 1 (plano) incorrecto: %+v", cands[1])
	}
}

func TestSelectCandidatePersistsReference(t *testing.T) {
	track := search.Result{ID: "vidsel", Title: "Numb", Uploader: "Linkin Park", Duration: 185}
	repo := newRepo(t, track)
	s := New(repo, http.DefaultClient)

	cand := Candidate{
		ProviderID: "111",
		Title:      "Numb",
		Artist:     "Linkin Park",
		Synced:     true,
		Query:      "numb linkin",
		body:       "[00:01.00]Encontrada",
	}
	l, err := s.SelectCandidate(context.Background(), track, cand)
	if err != nil {
		t.Fatalf("SelectCandidate err: %v", err)
	}
	if !l.Synced || len(l.Lines) != 1 || l.Lines[0].Text != "Encontrada" {
		t.Fatalf("letra seleccionada incorrecta: %+v", l)
	}

	entry, found, err := repo.Get(track.ID)
	if err != nil || !found {
		t.Fatalf("Get tras selección = (found=%v, err=%v)", found, err)
	}
	if entry.Query != "numb linkin" || entry.ProviderID != "111" {
		t.Fatalf("referencia no persistida: query=%q provider_id=%q", entry.Query, entry.ProviderID)
	}
	if entry.Body != "[00:01.00]Encontrada" || !entry.Synced {
		t.Fatalf("cuerpo/synced no persistido: %+v", entry)
	}
}

func TestFetchReusesSavedProviderID(t *testing.T) {
	var byIDHits, autoGetHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/get/999":
			atomic.AddInt32(&byIDHits, 1)
			_, _ = w.Write([]byte(`{"syncedLyrics":"[00:02.00]PorReferencia","plainLyrics":""}`))
		case "/api/get":
			atomic.AddInt32(&autoGetHits, 1)
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("ruta inesperada: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	track := search.Result{ID: "vidref", Title: "Song", Uploader: "Artist", Duration: 180}
	repo := newRepo(t, track)
	// Sembrar una referencia guardada SIN cuerpo (simula cuerpo perdido pero
	// referencia conservada): fromCache falla y debe entrar el reuso por provider_id.
	if err := repo.Upsert(storage.LyricsEntry{VideoID: track.ID, Body: "", ProviderID: "999"}); err != nil {
		t.Fatalf("seed reference: %v", err)
	}
	s := newService(t, repo, srv)

	l, err := s.Fetch(context.Background(), track, "Song", "Artist")
	if err != nil {
		t.Fatalf("Fetch err: %v", err)
	}
	if !l.Synced || len(l.Lines) != 1 || l.Lines[0].Text != "PorReferencia" {
		t.Fatalf("se esperaba letra resuelta por referencia guardada, got %+v", l)
	}
	if atomic.LoadInt32(&byIDHits) != 1 {
		t.Fatalf("se esperaba 1 golpe a /api/get/999, got %d", byIDHits)
	}
	if atomic.LoadInt32(&autoGetHits) != 0 {
		t.Fatalf("no debe usarse la consulta automática cuando hay referencia, hits=%d", autoGetHits)
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
	if _, err := s.Fetch(context.Background(), track, track.Title, track.Uploader); err != nil {
		t.Fatalf("primera Fetch: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("se esperaba 1 hit HTTP tras primera Fetch, got %d", got)
	}

	// Segunda llamada: debe resolverse desde la caché en BD sin HTTP.
	l, err := s.Fetch(context.Background(), track, track.Title, track.Uploader)
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
