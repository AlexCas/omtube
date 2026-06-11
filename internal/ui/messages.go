package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

// loadTrackCmd carga una pista en el reproductor.
func loadTrackCmd(p player.Player, track search.Result) tea.Cmd {
	return func() tea.Msg {
		err := p.Load(track.URL())
		return loadedMsg{track: track, err: err}
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

// fetchPositionCmd consulta posición/duración en un goroutine de Cmd, evitando
// bloquear el bucle Update con round-trips IPC.
func fetchPositionCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		pos, dur := p.Position()
		return posMsg{pos: pos, dur: dur}
	}
}
