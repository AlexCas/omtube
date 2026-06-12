package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// View renderiza la TUI.
func (m Model) View() string {
	if m.quitting {
		return "¡Hasta luego!\n"
	}

	if m.mode == modePicker {
		return m.picker.View()
	}

	if m.mode == modeLibrary || m.mode == modeCreatePlaylist {
		return m.renderLibrary()
	}

	var b strings.Builder
	b.WriteString(m.styles.title.Render("🎵 Omusic"))
	b.WriteString("\n\n")

	// Barra de búsqueda o estado.
	if m.mode == modeSearch {
		b.WriteString(m.input.View())
	} else {
		b.WriteString(m.styles.dim.Render(m.status))
	}
	b.WriteString("\n\n")

	// Paneles: resultados y cola lado a lado.
	results := m.renderResults()
	q := m.renderQueue()
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, results, q))
	b.WriteString("\n")

	// Paneles de enriquecimiento (letra/portada). Solo se dibujan cuando su
	// servicio está activo: con los toggles apagados la vista es la de la Fase 2.
	if enrich := m.renderEnrichment(); enrich != "" {
		b.WriteString(enrich)
		b.WriteString("\n")
	}

	// Now playing + progreso.
	b.WriteString(m.renderNowPlaying())
	b.WriteString("\n")
	b.WriteString(m.renderHelp())
	b.WriteString("\n")
	return m.center(b.String())
}

// center centra horizontalmente el bloque de la vista dentro del ancho de la
// terminal. Antes del primer WindowSizeMsg (m.width == 0) devuelve el contenido
// sin tocar para no colapsarlo contra el margen izquierdo.
func (m Model) center(s string) string {
	if m.width <= 0 {
		return s
	}
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, s)
}

func (m Model) renderResults() string {
	var b strings.Builder
	b.WriteString(m.styles.heading.Render("Resultados"))
	b.WriteString("\n")
	if len(m.results) == 0 {
		b.WriteString(m.styles.dim.Render("(vacío)"))
	}
	for i, r := range m.results {
		line := fmt.Sprintf("%s  %s", r.Title, m.styles.dim.Render(r.Uploader))
		if i == m.cursor {
			line = m.styles.selected.Render("➤ " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(m.cacheMark(r.ID) + truncate(line, 44))
		b.WriteString("\n")
	}
	return m.styles.panel.Width(48).Render(b.String())
}

func (m Model) renderQueue() string {
	var b strings.Builder
	b.WriteString(m.styles.heading.Render("Cola"))
	b.WriteString("\n")
	items := m.queue.Items()
	if len(items) == 0 {
		b.WriteString(m.styles.dim.Render("(vacía)"))
	}
	for i, r := range items {
		prefix := "  "
		line := r.Title
		if i == m.queue.Index() {
			prefix = m.styles.current.Render("▶ ")
			line = m.styles.current.Render(line)
		}
		b.WriteString(m.cacheMark(r.ID) + prefix + truncate(line, 28))
		b.WriteString("\n")
	}
	return m.styles.panel.Width(36).Render(b.String())
}

func (m Model) renderNowPlaying() string {
	cur, ok := m.queue.Current()
	if !ok {
		return m.styles.dim.Render("Nada en reproducción")
	}
	state := "▶"
	if m.player.Paused() {
		state = "⏸"
	}
	bar := progressBar(m.pos, m.dur, 30)
	return fmt.Sprintf("%s %s  %s  %s/%s  vol %d",
		state,
		m.styles.current.Render(truncate(cur.Title, 32)),
		bar,
		fmtTime(m.pos), fmtTime(m.dur),
		m.player.Volume(),
	)
}

func (m Model) renderHelp() string {
	return m.styles.help.Render(
		"/ buscar · enter encolar · espacio play/pausa · n/p sig/ant · +/- volumen · f favorito · a +playlist · L biblioteca · q salir")
}

// renderLibrary dibuja el modo biblioteca con sus tres secciones (playlists,
// favoritos, historial) y la sección activa resaltada.
func (m Model) renderLibrary() string {
	var b strings.Builder
	b.WriteString(m.styles.title.Render("📚 Biblioteca"))
	b.WriteString("\n\n")
	if m.mode == modeCreatePlaylist {
		b.WriteString(m.input.View())
	} else {
		b.WriteString(m.styles.dim.Render(m.status))
	}
	b.WriteString("\n\n")

	tabs := []struct {
		sec   librarySection
		label string
	}{
		{sectionPlaylists, "Playlists"},
		{sectionFavorites, "Favoritos"},
		{sectionHistory, "Historial"},
	}
	var head strings.Builder
	for i, t := range tabs {
		label := t.label
		if t.sec == m.libSection {
			label = m.styles.selected.Render("[" + label + "]")
		} else {
			label = m.styles.dim.Render(" " + label + " ")
		}
		head.WriteString(label)
		if i < len(tabs)-1 {
			head.WriteString("  ")
		}
	}
	b.WriteString(head.String())
	b.WriteString("\n\n")

	switch m.libSection {
	case sectionPlaylists:
		b.WriteString(m.renderLibList(playlistLines(m.libPlaylists), "(sin playlists)"))
	case sectionFavorites:
		b.WriteString(m.renderLibList(trackLines(m.libFavorites), "(sin favoritos)"))
	case sectionHistory:
		b.WriteString(m.renderLibList(trackLines(m.libHistory), "(historial vacío)"))
	}

	b.WriteString("\n")
	b.WriteString(m.styles.help.Render(
		"↑/↓ navegar · n/p sección · enter reproducir · f favorito · a +playlist · c crear playlist · esc/L volver · q salir"))
	b.WriteString("\n")
	return m.center(b.String())
}

// renderLibList dibuja una lista de líneas con el cursor de biblioteca, o el
// mensaje vacío si no hay elementos.
func (m Model) renderLibList(lines []string, empty string) string {
	if len(lines) == 0 {
		return m.styles.dim.Render(empty) + "\n"
	}
	var b strings.Builder
	for i, line := range lines {
		if i == m.libCursor {
			b.WriteString(m.styles.selected.Render("➤ " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func playlistLines(pls []storage.Playlist) []string {
	out := make([]string, 0, len(pls))
	for _, p := range pls {
		out = append(out, p.Name)
	}
	return out
}

func trackLines(tracks []search.Result) []string {
	out := make([]string, 0, len(tracks))
	for _, t := range tracks {
		line := t.Title
		if t.Uploader != "" {
			line += "  — " + t.Uploader
		}
		out = append(out, truncate(line, 60))
	}
	return out
}

// cacheMark devuelve un indicador de "cacheada" para la pista id, o dos espacios
// de alineación cuando no lo está o la caché está desactivada. Mantener un ancho
// fijo evita que las filas se descoloquen.
func (m Model) cacheMark(id string) string {
	if m.cache != nil && id != "" && m.cachedIDs[id] {
		return m.styles.current.Render("⤓ ")
	}
	return "  "
}

// renderEnrichment compone los paneles de letra y portada lado a lado. Devuelve
// "" cuando ninguno de los dos servicios está activo (paridad con la Fase 2).
func (m Model) renderEnrichment() string {
	hasLyrics := m.lyrics != nil
	hasArtwork := m.artwork != nil
	if !hasLyrics && !hasArtwork {
		return ""
	}
	var panels []string
	if hasLyrics {
		panels = append(panels, m.renderLyricsPanel())
	}
	if hasArtwork {
		panels = append(panels, m.renderArtworkPanel())
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

// renderLyricsPanel dibuja la letra de la pista actual; resalta la línea activa
// cuando es sincronizada y muestra "sin letra" cuando no hay ninguna.
func (m Model) renderLyricsPanel() string {
	var b strings.Builder
	b.WriteString(m.styles.heading.Render("Letra"))
	b.WriteString("\n")

	switch {
	case m.curLyrics.Empty():
		b.WriteString(m.styles.dim.Render("sin letra"))
	case m.curLyrics.Synced:
		b.WriteString(m.renderSyncedLyrics())
	default:
		b.WriteString(truncateLines(m.curLyrics.Plain, 48, 8))
	}
	return m.styles.panel.Width(50).Render(b.String())
}

// renderSyncedLyrics muestra una ventana de líneas alrededor de la línea activa,
// resaltándola.
func (m Model) renderSyncedLyrics() string {
	const window = 7
	lines := m.curLyrics.Lines
	if len(lines) == 0 {
		return m.styles.dim.Render("sin letra")
	}
	cur := m.lyricLine
	start := cur - window/2
	if start < 0 {
		start = 0
	}
	end := start + window
	if end > len(lines) {
		end = len(lines)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		text := truncate(lines[i].Text, 46)
		if i == cur {
			b.WriteString(m.styles.current.Render("▶ " + text))
		} else {
			b.WriteString("  " + m.styles.dim.Render(text))
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderArtworkPanel dibuja la portada renderizada de la pista actual, o un
// estado de degradación cuando no hay portada.
func (m Model) renderArtworkPanel() string {
	var b strings.Builder
	b.WriteString(m.styles.heading.Render("Portada"))
	b.WriteString("\n")
	if m.curArtwork == "" {
		b.WriteString(m.styles.dim.Render("[sin portada]"))
	} else {
		b.WriteString(m.curArtwork)
	}
	return m.styles.panel.Width(28).Render(b.String())
}

// truncateLines recorta un bloque de texto a maxLines líneas, cada una a maxCols
// columnas, para encajar en el panel sin desbordar.
func truncateLines(s string, maxCols, maxLines int) string {
	raw := strings.Split(s, "\n")
	if len(raw) > maxLines {
		raw = raw[:maxLines]
	}
	for i := range raw {
		raw[i] = truncate(raw[i], maxCols)
	}
	return strings.Join(raw, "\n")
}

func progressBar(pos, dur float64, width int) string {
	if dur <= 0 {
		return strings.Repeat("─", width)
	}
	filled := int(pos / dur * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("━", filled) + strings.Repeat("─", width-filled)
}

func fmtTime(secs float64) string {
	s := int(secs)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func truncate(s string, max int) string {
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
