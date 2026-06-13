package ui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/alexcasdev/terminaltube/internal/config"
	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/playlist"
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// fakePlayer es un reproductor de prueba que registra las cargas y expone un
// canal de eventos controlable, sin lanzar mpv.
type fakePlayer struct {
	events  chan player.Event
	loaded  []string
	tracks  []search.Result
	paused  bool
	volume  int
	pos, du float64
	stopped int
}

func newFakePlayer() *fakePlayer {
	return &fakePlayer{events: make(chan player.Event, 16), volume: 70}
}

func (f *fakePlayer) Load(src string) error { f.loaded = append(f.loaded, src); return nil }
func (f *fakePlayer) LoadTrack(src string, t search.Result) error {
	f.loaded = append(f.loaded, src)
	f.tracks = append(f.tracks, t)
	return nil
}
func (f *fakePlayer) TogglePause() error           { f.paused = !f.paused; return nil }
func (f *fakePlayer) Stop() error                  { f.stopped++; f.paused = false; return nil }
func (f *fakePlayer) AddVolume(d int) (int, error) { f.volume += d; return f.volume, nil }
func (f *fakePlayer) Position() (float64, float64) { return f.pos, f.du }
func (f *fakePlayer) Paused() bool                 { return f.paused }
func (f *fakePlayer) Volume() int                  { return f.volume }
func (f *fakePlayer) Events() <-chan player.Event  { return f.events }
func (f *fakePlayer) Close() error                 { return nil }

// fakeCache es una caché de prueba con un conjunto fijo de pistas "cacheadas".
type fakeCache struct {
	have      map[string]string // id -> ruta local
	downloads []string
}

func (c *fakeCache) Lookup(id string) (string, bool) { p, ok := c.have[id]; return p, ok }
func (c *fakeCache) Download(_ context.Context, r search.Result) (string, error) {
	c.downloads = append(c.downloads, r.ID)
	return "/tmp/" + r.ID + ".opus", nil
}

// fakeLyrics devuelve una letra fija y candidatos configurables.
type fakeLyrics struct {
	ly    lyrics.Lyrics
	cands []lyrics.Candidate
}

func (l fakeLyrics) Fetch(_ context.Context, _ search.Result, _, _ string) (lyrics.Lyrics, error) {
	return l.ly, nil
}

func (l fakeLyrics) Search(_ context.Context, _ string) ([]lyrics.Candidate, error) {
	return l.cands, nil
}

func (l fakeLyrics) SelectCandidate(_ context.Context, _ search.Result, c lyrics.Candidate) (lyrics.Lyrics, error) {
	return lyrics.Lyrics{Plain: "letra de " + c.Title}, nil
}

// fakeSearcher implementa Searcher, Resolver y PlaylistResolver para los flujos
// de ingesta por URL e importación.
type fakeSearcher struct {
	resolveTrack search.Result
	resolveErr   error
	plTracks     []search.Result
	plTitle      string
	plErr        error
}

func (f fakeSearcher) Search(_ context.Context, _ string, _ int) ([]search.Result, error) {
	return nil, nil
}
func (f fakeSearcher) Resolve(_ context.Context, _ string) (search.Result, error) {
	return f.resolveTrack, f.resolveErr
}
func (f fakeSearcher) ResolvePlaylist(_ context.Context, _ string) ([]search.Result, string, error) {
	return f.plTracks, f.plTitle, f.plErr
}

// fakeArtwork devuelve una portada fija.
type fakeArtwork struct{ art string }

func (a fakeArtwork) Render(_ context.Context, _ search.Result, _, _ int) string { return a.art }

// fakePresence registra las llamadas a Set/Clear.
type fakePresence struct {
	set   []string
	clear int
}

func (p *fakePresence) Set(title, artist string) { p.set = append(p.set, title+"|"+artist) }
func (p *fakePresence) Clear()                   { p.clear++ }

func newTestModel(t *testing.T, svc Services) Model {
	t.Helper()
	m := New(config.Config{}, nil, newFakePlayer(), nil, nil, nil, svc, zap.NewNop())
	m.width, m.height = 120, 40
	return m
}

func TestToggleOffParity_NoEnrichmentPanels(t *testing.T) {
	// Con todos los servicios apagados (nil), la vista no debe mostrar paneles de
	// letra ni portada: paridad con la Fase 2.
	m := newTestModel(t, Services{})
	m.curTrackID = "abc"
	m.queue.Add(search.Result{ID: "abc", Title: "Song"})
	out := m.View()
	if strings.Contains(out, "Letra") || strings.Contains(out, "Portada") {
		t.Fatalf("toggle-off debería ocultar paneles de enriquecimiento; got:\n%s", out)
	}
	if strings.Contains(out, "⤓") {
		t.Fatalf("toggle-off no debería mostrar indicador de caché; got:\n%s", out)
	}
}

func TestToggleOffParity_NoTrackChangeFanout(t *testing.T) {
	// Con servicios apagados, un EventTrackChange no debe abanicar Cmds extra.
	m := newTestModel(t, Services{})
	cmds := m.onTrackChange(search.Result{ID: "abc", Title: "Song"})
	if len(cmds) != 0 {
		t.Fatalf("esperaba 0 Cmds con toggles apagados, got %d", len(cmds))
	}
}

func TestLyricsPanel_NoLyricsState(t *testing.T) {
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}})
	out := m.renderLyricsPanel()
	if !strings.Contains(out, "sin letra") {
		t.Fatalf("esperaba estado 'sin letra'; got:\n%s", out)
	}
}

func TestLyricsPanel_SyncedHighlight(t *testing.T) {
	ly := lyrics.Lyrics{Synced: true, Lines: []lyrics.Line{
		{T: 0, Text: "linea uno"},
		{T: 10, Text: "linea dos"},
		{T: 20, Text: "linea tres"},
	}}
	m := newTestModel(t, Services{Lyrics: fakeLyrics{ly: ly}})
	m.curLyrics = ly
	m.pos = 12
	m.advanceLyric()
	if m.lyricLine != 1 {
		t.Fatalf("esperaba línea activa 1 en pos=12, got %d", m.lyricLine)
	}
	out := m.renderLyricsPanel()
	if !strings.Contains(out, "▶") || !strings.Contains(out, "linea dos") {
		t.Fatalf("esperaba resaltar 'linea dos'; got:\n%s", out)
	}
}

func TestLyricsPanel_PlainFallback(t *testing.T) {
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}})
	m.curLyrics = lyrics.Lyrics{Plain: "verso plano"}
	out := m.renderLyricsPanel()
	if !strings.Contains(out, "verso plano") {
		t.Fatalf("esperaba texto plano; got:\n%s", out)
	}
}

func TestArtworkPanel_RenderAndDegrade(t *testing.T) {
	m := newTestModel(t, Services{Artwork: fakeArtwork{}})
	// Sin portada renderizada: degradación.
	if out := m.renderArtworkPanel(); !strings.Contains(out, "[sin portada]") {
		t.Fatalf("esperaba degradación '[sin portada]'; got:\n%s", out)
	}
	// Con portada: se muestra el contenido renderizado.
	m.curArtwork = "ARTDATA"
	if out := m.renderArtworkPanel(); !strings.Contains(out, "ARTDATA") {
		t.Fatalf("esperaba portada renderizada; got:\n%s", out)
	}
}

func TestCacheIndicator(t *testing.T) {
	m := newTestModel(t, Services{Cache: &fakeCache{have: map[string]string{"abc": "/tmp/abc.opus"}}})
	m.cachedIDs["abc"] = true
	if mark := m.cacheMark("abc"); !strings.Contains(mark, "⤓") {
		t.Fatalf("esperaba indicador de caché para 'abc'; got %q", mark)
	}
	if mark := m.cacheMark("xyz"); strings.Contains(mark, "⤓") {
		t.Fatalf("no esperaba indicador para 'xyz'; got %q", mark)
	}
	// Sin caché (nil) nunca debe marcar.
	m2 := newTestModel(t, Services{})
	m2.cachedIDs["abc"] = true
	if mark := m2.cacheMark("abc"); strings.Contains(mark, "⤓") {
		t.Fatalf("caché desactivada no debe marcar; got %q", mark)
	}
}

func TestOnTrackChange_FanoutAndCacheLookup(t *testing.T) {
	// Pista ya cacheada: no debe encolar descarga, pero sí marcarse.
	fc := &fakeCache{have: map[string]string{"cached1": "/tmp/cached1.opus"}}
	fp := &fakePresence{}
	m := newTestModel(t, Services{
		Cache:    fc,
		Lyrics:   fakeLyrics{},
		Artwork:  fakeArtwork{},
		Presence: fp,
	})
	cmds := m.onTrackChange(search.Result{ID: "cached1", Title: "T", Uploader: "U"})
	// lyrics + artwork + presence = 3 Cmds (sin descarga porque ya está cacheada).
	if len(cmds) != 3 {
		t.Fatalf("esperaba 3 Cmds (sin descarga), got %d", len(cmds))
	}
	if !m.cachedIDs["cached1"] {
		t.Fatalf("la pista cacheada debería marcarse en cachedIDs")
	}
	if m.curTrackID != "cached1" {
		t.Fatalf("curTrackID no se actualizó: %q", m.curTrackID)
	}

	// Pista NO cacheada: debe encolar también la descarga ⇒ 4 Cmds.
	cmds = m.onTrackChange(search.Result{ID: "fresh", Title: "T2"})
	if len(cmds) != 4 {
		t.Fatalf("esperaba 4 Cmds (con descarga) para pista no cacheada, got %d", len(cmds))
	}
}

func TestPresenceClearedOnQueueFinished(t *testing.T) {
	// W2: al finalizar la cola (EventEndFile sin siguiente pista) se debe limpiar
	// la presencia de Discord, no solo al salir de la app.
	fp := &fakePresence{}
	m := newTestModel(t, Services{Presence: fp})
	// Cola con una sola pista ⇒ Next() es false ⇒ rama "cola finalizada".
	m.queue.Add(search.Result{ID: "abc", Title: "Song"})

	updated, _ := m.handlePlayerEvent(playerEventMsg{event: player.Event{Kind: player.EventEndFile}})
	if fp.clear != 1 {
		t.Fatalf("esperaba 1 llamada a Clear al finalizar la cola, got %d", fp.clear)
	}
	if !strings.Contains(updated.(Model).status, "finalizada") {
		t.Fatalf("esperaba estado 'cola finalizada'; got %q", updated.(Model).status)
	}
}

func TestPresenceNilSafeOnQueueFinished(t *testing.T) {
	// Sin presencia (toggle apagado), la rama de cola-finalizada no debe panicar.
	m := newTestModel(t, Services{})
	m.queue.Add(search.Result{ID: "abc", Title: "Song"})
	if _, _ = m.handlePlayerEvent(playerEventMsg{event: player.Event{Kind: player.EventEndFile}}); m.status == "" {
		// (no assertion más allá de no-panic; el estado se fija dentro del handler)
	}
}

func TestClearQueueStopsAndResets(t *testing.T) {
	fp := &fakePresence{}
	m := newTestModel(t, Services{Presence: fp})
	player := m.player.(*fakePlayer)
	m.queue.Add(search.Result{ID: "a", Title: "A"})
	m.queue.Add(search.Result{ID: "b", Title: "B"})
	m.started = true
	m.curTrackID = "a"
	m.curLyrics = lyrics.Lyrics{Plain: "x"}
	m.curArtwork = "ART"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	um := updated.(Model)

	if um.queue.Len() != 0 {
		t.Fatalf("cola no vaciada: Len=%d", um.queue.Len())
	}
	if player.stopped != 1 {
		t.Fatalf("esperaba 1 Stop al limpiar, got %d", player.stopped)
	}
	if um.started || um.curTrackID != "" || !um.curLyrics.Empty() || um.curArtwork != "" {
		t.Fatalf("estado de 'ahora suena' no reiniciado: %+v", um)
	}
	if fp.clear != 1 {
		t.Fatalf("esperaba limpiar presencia al limpiar la cola, got %d", fp.clear)
	}
	if !strings.Contains(um.status, "limpiada") {
		t.Fatalf("estado inesperado: %q", um.status)
	}
}

func TestURLResolvedEnqueuesAndExposesResult(t *testing.T) {
	m := newTestModel(t, Services{})
	track := search.Result{ID: "vid", Title: "Canción", Uploader: "Artista"}

	updated, cmd := m.Update(urlResolvedMsg{track: track})
	um := updated.(Model)

	if um.queue.Len() != 1 {
		t.Fatalf("la URL resuelta debería encolarse: Len=%d", um.queue.Len())
	}
	if len(um.results) != 1 || um.results[0].ID != "vid" {
		t.Fatalf("la pista resuelta debería exponerse como resultado: %+v", um.results)
	}
	if !um.started || cmd == nil {
		t.Fatal("primera pista debería arrancar la reproducción (cmd de carga)")
	}
	if !strings.Contains(um.status, "playlist") {
		t.Fatalf("el estado debería sugerir añadir a playlist: %q", um.status)
	}
}

func TestURLResolveErrorSurfaces(t *testing.T) {
	m := newTestModel(t, Services{})
	updated, _ := m.Update(urlResolvedMsg{err: context.DeadlineExceeded})
	if um := updated.(Model); !strings.Contains(um.status, "No se pudo resolver") {
		t.Fatalf("esperaba error de resolución en estado: %q", um.status)
	}
}

func TestPlaylistResolvedPromptsForName(t *testing.T) {
	m := newTestModel(t, Services{})
	tracks := []search.Result{{ID: "a"}, {ID: "b"}}

	updated, _ := m.Update(playlistResolvedMsg{tracks: tracks, title: "Mix"})
	um := updated.(Model)

	if um.mode != modeImportName {
		t.Fatalf("esperaba modeImportName, got %v", um.mode)
	}
	if len(um.importTracks) != 2 {
		t.Fatalf("esperaba 2 pistas pendientes de nombre, got %d", len(um.importTracks))
	}
}

func TestPlaylistResolvedEmptyDoesNotPrompt(t *testing.T) {
	m := newTestModel(t, Services{})
	updated, _ := m.Update(playlistResolvedMsg{tracks: nil})
	if um := updated.(Model); um.mode == modeImportName {
		t.Fatal("una playlist sin pistas no debería pedir nombre")
	}
}

func TestImportNameCreatesPlaylistWithTracks(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	pl := playlist.New(db.Playlists(), db.Tracks())

	m := New(config.Config{}, nil, newFakePlayer(), nil, pl, nil, Services{}, zap.NewNop())
	m.mode = modeImportName
	m.importTracks = []search.Result{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}
	m.input.SetValue("Mi Lista")

	updated, _ := m.updateImportNameMode(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.mode != modeNormal {
		t.Fatalf("tras crear, esperaba modeNormal, got %v", um.mode)
	}

	pls, err := pl.List()
	if err != nil || len(pls) != 1 || pls[0].Name != "Mi Lista" {
		t.Fatalf("playlist no creada correctamente: %+v err=%v", pls, err)
	}
	tracks, err := pl.Tracks(pls[0].ID)
	if err != nil || len(tracks) != 2 {
		t.Fatalf("esperaba 2 pistas en la playlist importada, got %d err=%v", len(tracks), err)
	}
}

func TestImportNameDuplicateStaysInPrompt(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	pl := playlist.New(db.Playlists(), db.Tracks())
	if _, err := pl.Create("Existente"); err != nil {
		t.Fatalf("Create previo: %v", err)
	}

	m := New(config.Config{}, nil, newFakePlayer(), nil, pl, nil, Services{}, zap.NewNop())
	m.mode = modeImportName
	m.importTracks = []search.Result{{ID: "a", Title: "A"}}
	m.input.SetValue("Existente")

	updated, _ := m.updateImportNameMode(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.mode != modeImportName {
		t.Fatalf("nombre duplicado debería mantener el prompt, got %v", um.mode)
	}
	if len(um.importTracks) != 1 {
		t.Fatal("las pistas pendientes deberían conservarse para reintentar")
	}
}

func TestLyricsSearchOpensPickerAndSelects(t *testing.T) {
	cands := []lyrics.Candidate{
		{ProviderID: "1", Title: "Numb", Artist: "Linkin Park", Synced: true},
		{ProviderID: "2", Title: "Numb (live)", Artist: "Linkin Park"},
	}
	m := newTestModel(t, Services{Lyrics: fakeLyrics{cands: cands}})
	m.lyricsTrack = search.Result{ID: "vid", Title: "Numb"}
	m.curTrackID = "vid"

	updated, _ := m.Update(lyricsCandidatesMsg{cands: cands})
	um := updated.(Model)
	if um.mode != modeLyricsPicker {
		t.Fatalf("esperaba modeLyricsPicker, got %v", um.mode)
	}

	// Seleccionar el primer candidato fija la letra (vía selectLyricsCmd → lyricsMsg).
	updated, cmd := um.updateLyricsPickerMode(tea.KeyMsg{Type: tea.KeyEnter})
	um = updated.(Model)
	if um.mode != modeNormal {
		t.Fatalf("tras seleccionar, esperaba modeNormal, got %v", um.mode)
	}
	if cmd == nil {
		t.Fatal("esperaba un cmd que fije la letra seleccionada")
	}
	msg := cmd()
	lm, ok := msg.(lyricsMsg)
	if !ok || lm.videoID != "vid" {
		t.Fatalf("esperaba lyricsMsg para la pista actual, got %#v", msg)
	}
	// El handler de lyricsMsg debe aplicar la letra a la pista actual.
	updated, _ = um.Update(lm)
	if got := updated.(Model).curLyrics.Plain; !strings.Contains(got, "Numb") {
		t.Fatalf("la letra seleccionada debería aplicarse al panel, got %q", got)
	}
}

func TestLyricsCandidatesEmptyShowsStatus(t *testing.T) {
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}})
	updated, _ := m.Update(lyricsCandidatesMsg{cands: nil})
	um := updated.(Model)
	if um.mode == modeLyricsPicker {
		t.Fatal("sin candidatos no debería abrir el picker")
	}
	if !strings.Contains(um.status, "Sin resultados") {
		t.Fatalf("esperaba 'Sin resultados de letra', got %q", um.status)
	}
}

func TestStaleEnrichmentDiscarded(t *testing.T) {
	// Una respuesta de letra que llega para una pista distinta a la actual se
	// descarta, evitando mostrar la letra de una pista anterior.
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}})
	m.curTrackID = "current"
	updated, _ := m.Update(lyricsMsg{videoID: "old", lyrics: lyrics.Lyrics{Plain: "vieja"}})
	if !updated.(Model).curLyrics.Empty() {
		t.Fatalf("una respuesta obsoleta no debería sobrescribir la letra actual")
	}
	updated, _ = updated.(Model).Update(lyricsMsg{videoID: "current", lyrics: lyrics.Lyrics{Plain: "actual"}})
	if updated.(Model).curLyrics.Plain != "actual" {
		t.Fatalf("una respuesta de la pista actual debería aplicarse")
	}
}
