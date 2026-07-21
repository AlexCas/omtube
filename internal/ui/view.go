package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// breakpoint clasifica el ancho de la terminal en las tres bandas responsivas
// del rediseño Caelestia.
type breakpoint int

const (
	bpNarrow breakpoint = iota // < 90 cols
	bpMedium                   // 90–119 cols
	bpWide                     // >= 120 cols
)

// classify devuelve el breakpoint correspondiente a un ancho de terminal.
func classify(width int) breakpoint {
	switch {
	case width < 90:
		return bpNarrow
	case width < 120:
		return bpMedium
	default:
		return bpWide
	}
}

// layout agrupa las dimensiones derivadas del tamaño de la terminal que usan
// los render helpers. Los anchos de panel (queueW/lyricsW/artW) son el valor
// que se pasa a Style.Width(); el borde redondeado añade 2 columnas por panel,
// ya descontadas del presupuesto en computeLayout.
type layout struct {
	bp                      breakpoint
	queueW, lyricsW, artW   int // anchos de panel (parámetro de Width)
	progressW               int // ancho de la barra de progreso
	bodyH                   int // filas disponibles para la sección media
	maxQueueRows            int // filas visibles de la cola
	lyricWindow, plainLines int // ventana de letra sincronizada / plana
	nowTitleTrunc           int // truncado del título en "ahora suena"
	libLineTrunc            int // truncado de líneas de biblioteca
	showArtwork             bool
}

// minUsable es el piso del ancho útil: por debajo el layout no intenta
// comprimirse más (terminales tan estrechas no están soportadas).
const minUsable = 40

// panelBorder son las columnas que el borde redondeado añade fuera de Width().
const panelBorder = 2

// helpMainText es la línea de ayuda del modo principal. Vive como constante
// para que computeLayout pueda medir cuántas filas ocupa envuelta al ancho
// actual (forma parte del chrome vertical de la vista).
const helpMainText = "/ buscar · u URL · i importar · enter encolar · espacio play/pausa · n/p sig/ant · y letra · C limpiar · f favorito · a +playlist · L biblioteca · q salir"

// helpRows mide cuántas filas ocupa la línea de ayuda envuelta al ancho útil
// de la terminal, con la misma regla de envoltura que wrapHelp.
func helpRows(width int) int {
	maxW := width - 2
	if maxW <= 0 || lipgloss.Width(helpMainText) <= maxW {
		return 1
	}
	return lipgloss.Height(lipgloss.NewStyle().Width(maxW).Render(helpMainText))
}

// computeLayout deriva el layout del tamaño actual de la terminal. Es una
// función pura: mismo (width, height) ⇒ mismo layout. El ancho gobierna el
// breakpoint y las columnas; el alto dimensiona la sección media (bodyH) y
// de ella derivan las filas de cola y las ventanas de letra.
func computeLayout(width, height int) layout {
	bp := classify(width)
	usable := max(width-2, minUsable) // margen exterior de 2 columnas

	// Chrome vertical medido sobre View(): título con borde (3), separador (1),
	// ahora-suena (1), separador (1), estado/búsqueda (1), separador (1) antes
	// de la sección media; y separador (1), ayuda (envuelta, medida aparte),
	// visualizador (1) y salto final (1) después de ella. Total fijo: 11 filas
	// más las que ocupe la ayuda al ancho actual.
	const chromeFixed = 11
	// minBody es el piso de la sección media: por debajo los paneles ya no se
	// comprimen más y se acepta que la vista exceda alturas extremas.
	const minBody = 4
	bodyH := max(height-(chromeFixed+helpRows(width)), minBody)

	var queueW, lyricsW, artW int
	if bp == bpNarrow {
		// Dos columnas: la portada se oculta bajo el breakpoint narrow y el
		// presupuesto se reparte entre cola (~42%) y letra (~58%).
		budget := usable - 2*panelBorder
		queueW = clamp(int(math.Round(float64(budget)*0.42)), 24, 40)
		lyricsW = budget - queueW
	} else {
		// Tres columnas con borde. Proporciones por breakpoint: medium
		// 34/40/26, wide 30/44/26; el remanente de redondeo se pliega en la
		// letra para que la fila central nunca exceda usable.
		budget := usable - 3*panelBorder
		qPct, aPct := 0.34, 0.26
		if bp == bpWide {
			qPct = 0.30
		}
		const qMin, lMin, aMin, aMax = 24, 28, 24, 28
		queueW = clamp(int(math.Round(float64(budget)*qPct)), qMin, 40)
		artW = clamp(int(math.Round(float64(budget)*aPct)), aMin, aMax)
		lyricsW = budget - queueW - artW
		if lyricsW < lMin {
			// Devolver columnas desde cola y portada, sin bajar de sus mínimos.
			if give := min(lMin-lyricsW, queueW-qMin); give > 0 {
				queueW -= give
				lyricsW += give
			}
			if give := min(lMin-lyricsW, artW-aMin); give > 0 {
				artW -= give
				lyricsW += give
			}
		}
	}

	// Línea "ahora suena": el resto de la línea (estado, tiempos, volumen y
	// separadores) ocupa ~24 columnas fijas; título y barra reparten el resto.
	const nowDecor = 24
	nowTitleTrunc := max(8, lyricsW-4)
	progressW := clamp(width-nowDecor-nowTitleTrunc, 8, 40)

	// Filas de la cola: bodyH menos el chrome real del panel (2 de borde,
	// 1 de encabezado y hasta 2 marcadores ▲/▼), con piso 3 para no colapsar
	// en alturas chicas. El techo queda en 10 (la ventana histórica): la letra
	// es la región que crece primero con el alto.
	maxQueueRows := clamp(bodyH-5, 3, 10)
	// Ventana de letra sincronizada: bodyH menos borde (2) y encabezado (1),
	// normalizada a impar para que la línea activa quede centrada.
	lyricWindow := clamp(bodyH-3, 3, 12)
	if lyricWindow%2 == 0 {
		lyricWindow--
	}

	return layout{
		bp:            bp,
		queueW:        queueW,
		lyricsW:       lyricsW,
		artW:          artW,
		progressW:     progressW,
		bodyH:         bodyH,
		maxQueueRows:  maxQueueRows,
		lyricWindow:   lyricWindow,
		plainLines:    clamp(bodyH-3, 3, 12),
		nowTitleTrunc: nowTitleTrunc,
		libLineTrunc:  max(20, width-4),
		showArtwork:   bp != bpNarrow,
	}
}

// clamp acota v al rango [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// View renderiza la TUI.
func (m Model) View() string {
	if m.quitting {
		return "¡Hasta luego!\n"
	}

	if m.mode == modePicker || m.mode == modeLyricsPicker {
		return m.picker.View()
	}

	// modeResults: modal de pantalla completa; la vista principal queda oculta.
	if m.mode == modeResults {
		var rb strings.Builder
		rb.WriteString(m.resultsList.View())
		rb.WriteString("\n")
		rb.WriteString(m.styles.help.Render(
			"enter encolar · a +playlist · f favorito · ↑/↓ navegar · esc cerrar"))
		return rb.String()
	}

	// Layout derivado del tamaño actual: se calcula una vez por render y se
	// enhebra en cada helper (anchos, truncados y ventanas fluidos).
	l := computeLayout(m.width, m.height)

	if m.mode == modeLibrary || m.mode == modeCreatePlaylist {
		return m.renderLibrary(l)
	}

	var b strings.Builder
	b.WriteString(m.styles.title.Render("🎵 Omusic"))
	b.WriteString("\n\n")

	// Barra de "ahora suena" en la parte superior (Caelestia layout).
	b.WriteString(m.renderNowPlaying(l))
	b.WriteString("\n\n")

	// Barra de búsqueda/prompt o estado.
	if m.isInputMode() {
		b.WriteString(m.input.View())
	} else {
		b.WriteString(m.styles.dim.Render(m.status))
	}
	b.WriteString("\n\n")

	// Sección media: cola + paneles de enriquecimiento (letra/portada) lado a
	// lado. Los paneles de enriquecimiento solo se dibujan cuando su servicio
	// está activo; con los toggles apagados la vista es la de la Fase 2.
	b.WriteString(m.renderMiddleSection(l))
	b.WriteString("\n\n")

	// Ayuda en la parte inferior, seguida del visualizador de barras.
	help := m.renderHelp()
	b.WriteString(help)
	b.WriteString("\n")
	// Visualizador de barras: cubre exactamente el ancho de la línea de
	// instrucciones y se anima mientras suena la música.
	b.WriteString(m.renderVisualizer(lipgloss.Width(help)))
	b.WriteString("\n")
	return m.center(b.String())
}

// barLevels son los caracteres de bloque ordenados de menor a mayor altura, que
// componen cada columna del visualizador.
var barLevels = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// renderVisualizer dibuja una fila de barras de ancho width a modo de
// visualizador decorativo. Mientras hay reproducción las alturas se mueven con
// animFrame; en pausa o sin pista quedan planas (nivel mínimo).
func (m Model) renderVisualizer(width int) string {
	if width <= 0 {
		return ""
	}
	playing := m.isPlaying()
	var b strings.Builder
	b.Grow(width * 3)
	for col := 0; col < width; col++ {
		level := 0
		if playing {
			level = barLevel(col, m.animFrame)
		}
		b.WriteRune(barLevels[level])
	}
	return m.styles.viz.Render(b.String())
}

// barLevel calcula la altura (0..len(barLevels)-1) de la columna col en el frame
// f combinando dos ondas senoidales desfasadas, para un movimiento tipo
// ecualizador. Es determinista: igual (col, f) ⇒ igual altura.
func barLevel(col, f int) int {
	levels := len(barLevels)
	x, t := float64(col), float64(f)
	v := math.Sin(x*0.45+t*0.30) + 0.6*math.Sin(x*0.17-t*0.21) // v ∈ [-1.6, 1.6]
	n := int(math.Round((v + 1.6) / 3.2 * float64(levels-1)))
	if n < 0 {
		n = 0
	}
	if n > levels-1 {
		n = levels - 1
	}
	return n
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

// renderQueue conserva la firma histórica (la invocan pruebas del paquete);
// deriva el layout del tamaño actual del modelo.
func (m Model) renderQueue() string {
	return m.renderQueueAt(computeLayout(m.width, m.height))
}

// renderQueueAt dibuja el panel de cola con las dimensiones del layout. Una
// cola más larga que l.maxQueueRows (p.ej. una playlist importada) se muestra
// como una ventana deslizante alrededor de la pista actual, evitando que el
// panel crezca sin límite y rompa el layout. La cola interna se mantiene
// completa. Cada fila descuenta 6 columnas del ancho del panel: 2 de padding,
// 2 de marca de caché y 2 del prefijo ▶/espacios.
func (m Model) renderQueueAt(l layout) string {
	var b strings.Builder
	items := m.queue.Items()
	total := len(items)
	heading := "Cola"
	if total > 0 {
		heading = fmt.Sprintf("Cola (%d)", total)
	}
	b.WriteString(m.styles.heading.Render(heading))
	b.WriteString("\n")
	if total == 0 {
		b.WriteString(m.styles.dim.Render("(vacía)"))
		return m.styles.panel.Width(l.queueW).Render(b.String())
	}

	start, end := queueWindow(m.queue.Index(), total, l.maxQueueRows)
	if start > 0 {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  ▲ %d más", start)))
		b.WriteString("\n")
	}
	for i := start; i < end; i++ {
		r := items[i]
		prefix := "  "
		line := r.Title
		if i == m.queue.Index() {
			prefix = m.styles.current.Render("▶ ")
			line = m.styles.current.Render(line)
		}
		b.WriteString(m.cacheMark(r.ID) + prefix + truncate(line, l.queueW-6))
		b.WriteString("\n")
	}
	if end < total {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  ▼ %d más", total-end)))
	}
	return m.styles.panel.Width(l.queueW).Render(b.String())
}

// queueWindow calcula el rango [start, end) de pistas a mostrar para una cola de
// total elementos con la actual en idx, limitado a window filas. Mantiene la
// actual visible con un pequeño contexto previo, de modo que al avanzar la
// reproducción la ventana se desliza y las próximas pistas quedan a la vista.
func queueWindow(idx, total, window int) (start, end int) {
	if total <= window {
		return 0, total
	}
	if idx < 0 {
		idx = 0
	}
	const lead = 2 // pistas ya pasadas que se conservan como contexto encima
	start = idx - lead
	if start < 0 {
		start = 0
	}
	end = start + window
	if end > total {
		end = total
		start = end - window
	}
	return start, end
}

func (m Model) renderNowPlaying(l layout) string {
	cur, ok := m.queue.Current()
	if !ok {
		return m.styles.dim.Render("Nada en reproducción")
	}
	state := "▶"
	if m.player.Paused() {
		state = "⏸"
	}
	bar := progressBar(m.pos, m.dur, l.progressW)
	return fmt.Sprintf("%s %s  %s  %s/%s  vol %d",
		state,
		m.styles.current.Render(truncate(cur.Title, l.nowTitleTrunc)),
		bar,
		fmtTime(m.pos), fmtTime(m.dur),
		m.player.Volume(),
	)
}

// isInputMode indica si el modo actual usa el input de texto compartido (búsqueda
// o cualquiera de los prompts de URL/importación/letra), para dibujarlo en la vista.
func (m Model) isInputMode() bool {
	switch m.mode {
	case modeSearch, modeURLInput, modeImportURL, modeImportName, modeLyricsSearch:
		return true
	}
	return false
}

func (m Model) renderHelp() string {
	return m.wrapHelp(helpMainText)
}

// wrapHelp aplica el estilo de ayuda y, cuando el texto no cabe en el ancho
// útil de la terminal, lo envuelve para que ninguna línea desborde.
func (m Model) wrapHelp(text string) string {
	if maxW := m.width - 2; maxW > 0 && lipgloss.Width(text) > maxW {
		return m.styles.help.Width(maxW).Render(text)
	}
	return m.styles.help.Render(text)
}

// renderLibrary dibuja el modo biblioteca con sus tres secciones (playlists,
// favoritos, historial) y la sección activa resaltada.
func (m Model) renderLibrary(l layout) string {
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
		b.WriteString(m.renderLibList(trackLines(m.libFavorites, l.libLineTrunc), "(sin favoritos)"))
	case sectionHistory:
		b.WriteString(m.renderLibList(trackLines(m.libHistory, l.libLineTrunc), "(historial vacío)"))
	}

	b.WriteString("\n")
	b.WriteString(m.wrapHelp(
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

func trackLines(tracks []search.Result, maxCols int) []string {
	out := make([]string, 0, len(tracks))
	for _, t := range tracks {
		line := t.Title
		if t.Uploader != "" {
			line += "  — " + t.Uploader
		}
		out = append(out, truncate(line, maxCols))
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

// renderMiddleSection compone la cola y los paneles de enriquecimiento en una
// fila horizontal. Cuando los servicios de enriquecimiento están apagados (nil)
// solo se muestra la cola, manteniendo paridad con la Fase 2. La banda ocupa
// las l.bodyH filas de la sección media: PlaceVertical rellena el alto sobrante
// (alineada arriba) y deja el contenido intacto cuando ya es más alto, de modo
// que ningún elemento obligatorio se recorta; en alturas chicas los paneles ya
// llegan encogidos por los clamps de computeLayout.
func (m Model) renderMiddleSection(l layout) string {
	band := m.renderQueueAt(l)
	if enrich := m.renderEnrichment(l); enrich != "" {
		band = lipgloss.JoinHorizontal(lipgloss.Top, band, enrich)
	}
	return lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)
}

// renderEnrichment compone los paneles de letra y portada lado a lado. Devuelve
// "" cuando ninguno de los dos servicios está activo (paridad con la Fase 2).
// Bajo el breakpoint narrow la portada se oculta (no se encoge ni se mueve),
// dejando solo cola + letra en dos columnas.
func (m Model) renderEnrichment(l layout) string {
	hasLyrics := m.lyrics != nil
	hasArtwork := m.artwork != nil && l.showArtwork
	if !hasLyrics && !hasArtwork {
		return ""
	}
	var panels []string
	if hasLyrics {
		panels = append(panels, m.renderLyricsPanelAt(l))
	}
	if hasArtwork {
		panels = append(panels, m.renderArtworkPanelAt(l))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

// renderLyricsPanel conserva la firma histórica (la invocan pruebas del
// paquete); deriva el layout del tamaño actual del modelo.
func (m Model) renderLyricsPanel() string {
	return m.renderLyricsPanelAt(computeLayout(m.width, m.height))
}

// renderLyricsPanelAt dibuja la letra de la pista actual; resalta la línea
// activa cuando es sincronizada y muestra "sin letra" cuando no hay ninguna.
func (m Model) renderLyricsPanelAt(l layout) string {
	var b strings.Builder
	b.WriteString(m.styles.heading.Render("Letra"))
	b.WriteString("\n")

	switch {
	case m.curLyrics.Empty():
		b.WriteString(m.styles.dim.Render("sin letra"))
	case m.curLyrics.Synced:
		b.WriteString(m.renderSyncedLyrics(l))
	default:
		b.WriteString(truncateLines(m.curLyrics.Plain, l.lyricsW-2, l.plainLines))
	}
	return m.styles.panel.Width(l.lyricsW).Render(b.String())
}

// renderSyncedLyrics muestra una ventana de líneas alrededor de la línea activa,
// resaltándola. El truncado descuenta 4 columnas del ancho del panel: 2 de
// padding y 2 del prefijo "▶ ".
func (m Model) renderSyncedLyrics(l layout) string {
	window := l.lyricWindow
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
		text := truncate(lines[i].Text, l.lyricsW-4)
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

// renderArtworkPanel conserva la firma histórica (la invocan pruebas del
// paquete); deriva el layout del tamaño actual del modelo.
func (m Model) renderArtworkPanel() string {
	return m.renderArtworkPanelAt(computeLayout(m.width, m.height))
}

// renderArtworkPanelAt dibuja la portada renderizada de la pista actual, o un
// estado de degradación cuando no hay portada.
func (m Model) renderArtworkPanelAt(l layout) string {
	var b strings.Builder
	b.WriteString(m.styles.heading.Render("Portada"))
	b.WriteString("\n")
	if m.curArtwork == "" {
		b.WriteString(m.styles.dim.Render("[sin portada]"))
	} else {
		b.WriteString(m.curArtwork)
	}
	return m.styles.panel.Width(l.artW).Render(b.String())
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
