package ui

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/alexcasdev/terminaltube/internal/config"
	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/search"
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

// fakeLyrics devuelve una letra fija.
type fakeLyrics struct{ ly lyrics.Lyrics }

func (l fakeLyrics) Fetch(_ context.Context, _, _, _ string, _ int) (lyrics.Lyrics, error) {
	return l.ly, nil
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
