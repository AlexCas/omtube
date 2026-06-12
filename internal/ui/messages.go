package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/metadata"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/search"
)

// searchResultsMsg transporta el resultado de una búsqueda asíncrona.
type searchResultsMsg struct {
	results []search.Result
	err     error
}

// loadedMsg indica el resultado de cargar una pista.
type loadedMsg struct {
	track search.Result
	err   error
}

// playerEventMsg envuelve un evento emitido por el reproductor.
type playerEventMsg struct{ event player.Event }

// tickMsg dispara el refresco de la barra de progreso.
type tickMsg time.Time

// animTickMsg dispara el refresco del visualizador de barras (más frecuente que
// el tick de progreso para que la animación se vea fluida).
type animTickMsg time.Time

// posMsg transporta la posición/duración consultadas fuera del bucle Update.
type posMsg struct{ pos, dur float64 }

// doSearchCmd ejecuta una búsqueda en segundo plano.
func doSearchCmd(s search.Searcher, q string, n int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := s.Search(ctx, q, n)
		return searchResultsMsg{results: res, err: err}
	}
}

// lyricsMsg transporta la letra resuelta para una pista (vacía ⇒ "sin letra").
type lyricsMsg struct {
	videoID string
	lyrics  lyrics.Lyrics
}

// artworkMsg transporta la portada renderizada para una pista (cadena lista para
// pintar: secuencia de escape o placeholder).
type artworkMsg struct {
	videoID string
	art     string
}

// cacheDoneMsg indica que una pista terminó de descargarse a la caché local.
type cacheDoneMsg struct {
	videoID string
	err     error
}

// loadTrackCmd resuelve la fuente (archivo cacheado si existe, si no la URL de
// YouTube) y carga la pista en el reproductor emitiendo el cambio de pista.
func loadTrackCmd(p player.Player, c cacheService, track search.Result) tea.Cmd {
	return func() tea.Msg {
		src := track.URL()
		if c != nil {
			if path, ok := c.Lookup(track.ID); ok {
				src = path
			}
		}
		err := p.LoadTrack(src, track)
		return loadedMsg{track: track, err: err}
	}
}

// fetchLyricsCmd obtiene la letra de la pista en segundo plano. Es un no-op
// (sin mensaje) cuando el servicio está deshabilitado (nil).
func fetchLyricsCmd(l lyricsService, track search.Result) tea.Cmd {
	if l == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		// Normaliza (artist, title) solo para la consulta saliente; los campos
		// crudos de track quedan intactos (se conservan para mostrar y para el
		// ID de caché).
		artist, title := metadata.Normalize(track)
		ly, _ := l.Fetch(ctx, track, title, artist)
		return lyricsMsg{videoID: track.ID, lyrics: ly}
	}
}

// renderArtworkCmd renderiza la portada de la pista en segundo plano. Es un
// no-op cuando el servicio está deshabilitado (nil).
func renderArtworkCmd(a artworkService, track search.Result, w, h int) tea.Cmd {
	if a == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		art := a.Render(ctx, track, w, h)
		return artworkMsg{videoID: track.ID, art: art}
	}
}

// setPresenceCmd publica la pista actual como presencia de Discord (fire and
// forget). Es un no-op cuando el servicio está deshabilitado (nil).
func setPresenceCmd(p presenceService, track search.Result) tea.Cmd {
	if p == nil {
		return nil
	}
	return func() tea.Msg {
		p.Set(track.Title, track.Uploader)
		return nil
	}
}

// cacheDownloadCmd descarga la pista a la caché local en segundo plano. Es un
// no-op cuando la caché está deshabilitada (nil).
func cacheDownloadCmd(c cacheService, track search.Result) tea.Cmd {
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_, err := c.Download(ctx, track)
		return cacheDoneMsg{videoID: track.ID, err: err}
	}
}

// waitForEventCmd espera el siguiente evento del reproductor y se re-encola.
func waitForEventCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		return playerEventMsg{event: <-p.Events()}
	}
}

// tickCmd programa el siguiente refresco de progreso.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// animTickCmd programa el siguiente frame del visualizador (~8 fps).
func animTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return animTickMsg(t) })
}

// fetchPositionCmd consulta posición/duración en un goroutine de Cmd, evitando
// bloquear el bucle Update con round-trips IPC.
func fetchPositionCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		pos, dur := p.Position()
		return posMsg{pos: pos, dur: dur}
	}
}
