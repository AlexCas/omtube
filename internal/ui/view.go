package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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
// los render helpers. La división principal es sidebar | main (design D1):
// dos cajas de altura completa cuyos anchos exteriores son sidebarW/mainW.
// Los anchos de panel (queueW/lyricsW/artW) son el valor que se pasa a
// Style.Width(); el borde redondeado añade 2 columnas por panel, ya
// descontadas del presupuesto en computeLayout.
type layout struct {
	bp                      breakpoint
	sidebarW, mainW         int  // anchos exteriores de las dos columnas (parámetro de Width)
	sidebarH, mainH         int  // = bodyH; alto que llena cada columna
	slimRail                bool // true en narrow (<90): sidebar colapsada a rail
	queueW, lyricsW, artW   int  // anchos de panel (parámetro de Width)
	progressW               int  // ancho de la barra de progreso
	bodyH                   int  // filas disponibles para la sección media
	maxQueueRows            int  // filas visibles de la cola
	lyricWindow, plainLines int  // ventana de letra sincronizada / plana
	nowTitleTrunc           int  // truncado del título en "ahora suena"
	libLineTrunc            int  // truncado de líneas de biblioteca (interior de la caja main)
	showArtwork             bool
	navRows                 int  // filas del bloque nav de la sidebar (0 cuando no cabe)
	compactChrome           bool // alturas mínimas: tarjeta de 1 fila y sin separadores del cuerpo
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

	// Chrome vertical medido sobre View() (design D5a): título con borde (3),
	// separador (1), estado (1), separador (1); tras el cuerpo: separador (1),
	// tarjeta de "ahora suena" (4 = 2 borde + 2 contenido), separador (1),
	// ayuda (medida aparte), visualizador (1) y salto final (1). Re-medido en
	// el slice 2: la barra superior (-2) se vuelve tarjeta al pie (+5) ⇒
	// 11 - 2 + 5 = 14 filas fijas más las de la ayuda.
	const chromeFixed = 14
	// En alturas mínimas el chrome cede antes que los elementos obligatorios:
	// la tarjeta colapsa a una fila de contenido (-1) y se omiten los dos
	// separadores del cuerpo (-2) ⇒ 11 fijas, de modo que a 60×20 sigan
	// cabiendo encabezado de cola + 3 filas de pista (Element Parity) sin que
	// la vista exceda la terminal.
	const chromeCompact = 11
	// minBody es el piso duro de la sección media; minFullBody es el cuerpo
	// mínimo con encabezado + 3 filas de cola en la sidebar: por debajo se
	// activa el chrome compacto.
	const minBody = 4
	const minFullBody = 6
	hr := helpRows(width)
	bodyH := height - (chromeFixed + hr)
	compactChrome := bodyH < minFullBody
	if compactChrome {
		bodyH = height - (chromeCompact + hr)
	}
	bodyH = max(bodyH, minBody)

	// División principal sidebar | main (design D1): dos cajas con borde de
	// altura completa; cada una cuesta panelBorder columnas fuera de Width().
	// Invariante D1d: sidebarW + mainW + 2*panelBorder == usable, por
	// construcción (mainW toma el remanente del presupuesto).
	split := usable - 2*panelBorder
	slimRail := bp == bpNarrow
	var sidebarW int
	if slimRail {
		// Rail delgado (<90, design D1c): la cola se comprime a un rail y el
		// área main conserva el ancho máximo disponible para la letra.
		const railMin, railMax = 16, 22
		sidebarW = clamp(int(math.Round(float64(split)*0.22)), railMin, railMax)
	} else {
		const sbMin, sbMax = 26, 40
		sidebarW = clamp(int(math.Round(float64(split)*0.30)), sbMin, sbMax)
	}
	mainW := split - sidebarW

	// Anchos interiores del área main (design D1f): portada y letra se APILAN
	// a ancho completo dentro de la caja main (slice 2), así que ambos usan
	// mainW como columna única; bajo narrow la portada se oculta.
	showArtwork := bp != bpNarrow
	queueW := sidebarW
	lyricsW := mainW
	artW := 0
	if showArtwork {
		artW = mainW
	}

	// Contenido de la tarjeta "ahora suena": el resto de la fila (estado,
	// tiempos, volumen y separadores) ocupa ~24 columnas fijas; título y barra
	// reparten el ancho de texto interior de la tarjeta (usable menos borde y
	// padding), de modo que la fila compacta nunca envuelve dentro de la caja.
	const nowDecor = 24
	cardText := usable - panelBorder - 2
	nowTitleTrunc := max(8, min(lyricsW-4, cardText-nowDecor-8))
	progressW := clamp(cardText-nowDecor-nowTitleTrunc, 8, 40)

	// Alturas de las columnas (design D2a): ambas llenan la sección media.
	sidebarH := bodyH
	mainH := bodyH

	// Bloque de navegación de la sidebar (design D4a/D2e): 4 ítems + 1 regla
	// acentuada = 5 filas. Solo se dibuja cuando cabe junto al encabezado y
	// al menos 3 filas de cola (5 nav + 5 chrome de cola + 3 filas): la nav
	// cede antes que la ventana de cola (Element Parity a 60×20).
	navRows := 0
	if sidebarH >= 5+5+3 {
		navRows = 5
	}
	// Filas de la cola (design D2b/D2e): sidebarH menos el chrome real del
	// panel (2 de borde, 1 de encabezado, hasta 2 marcadores ▲/▼ y el bloque
	// nav cuando está presente). Sin techo fijo, con techo duro en el interior
	// de la caja (fillBoxHeight).
	queueChrome := 5 + navRows
	maxQueueRows := clamp(sidebarH-queueChrome, 3, max(sidebarH-panelBorder-navRows-1, 3))
	// Ventana de letra sincronizada (design D2c/D2f): mainH menos el chrome
	// del apilado — borde de la caja main (2) + encabezado de letra (1), más
	// el bloque de portada (encabezado 1 + 12 filas de arte) cuando se
	// muestra —, sin techo fijo y normalizada a impar para que la línea
	// activa quede centrada.
	lyricChrome := 3
	if showArtwork {
		lyricChrome += 13
	}
	lyricWindow := clamp(mainH-lyricChrome, 3, mainH)
	if lyricWindow%2 == 0 {
		lyricWindow--
	}
	// Letra plana (design D2d): misma derivación sin normalización impar.
	plainLines := clamp(mainH-lyricChrome, 3, mainH)

	return layout{
		bp:            bp,
		sidebarW:      sidebarW,
		mainW:         mainW,
		sidebarH:      sidebarH,
		mainH:         mainH,
		slimRail:      slimRail,
		queueW:        queueW,
		lyricsW:       lyricsW,
		artW:          artW,
		progressW:     progressW,
		bodyH:         bodyH,
		maxQueueRows:  maxQueueRows,
		lyricWindow:   lyricWindow,
		plainLines:    plainLines,
		nowTitleTrunc: nowTitleTrunc,
		libLineTrunc:  max(10, mainW-4),
		showArtwork:   showArtwork,
		navRows:       navRows,
		compactChrome: compactChrome,
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

// themedList devuelve una copia tematizada de un list.Model con el delegate
// Caelestia y una barra de título sin fondo: el Styles.Title por defecto de
// bubbles/list trae Background("62"), que aquí se reemplaza por un estilo
// translúcido con foreground mauve. Opera sobre la copia por valor y no muta
// el estado del modelo.
func themedList(l list.Model) list.Model {
	l.SetDelegate(caelestiaListDelegate())
	s := l.Styles
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#e0aaff")).
		Padding(0, 1)
	l.Styles = s
	return l
}

// View renderiza la TUI.
func (m Model) View() string {
	if m.quitting {
		return "¡Hasta luego!\n"
	}

	// Pickers de letra/lista: a pantalla completa por diseño (design D8) —
	// los maneja bubbles/list desde Update y no entran en la caja main.
	if m.mode == modePicker || m.mode == modeLyricsPicker {
		return themedList(m.picker).View()
	}

	// modeResults: modal de pantalla completa; la vista principal queda oculta.
	if m.mode == modeResults {
		var rb strings.Builder
		rb.WriteString(themedList(m.resultsList).View())
		rb.WriteString("\n")
		rb.WriteString(m.styles.help.Render(
			"enter encolar · a +playlist · f favorito · ↑/↓ navegar · esc cerrar"))
		return rb.String()
	}

	// Layout derivado del tamaño actual: se calcula una vez por render y se
	// enhebra en cada helper (anchos, truncados y ventanas fluidos).
	// El modo biblioteca (modeLibrary/modeCreatePlaylist) ya no retorna una
	// vista aparte: fluye por este mismo bloque y renderMain enruta su
	// contenido al área main con la sidebar persistente (design D8).
	l := computeLayout(m.width, m.height)

	var b strings.Builder
	b.WriteString(m.styles.title.Render("🎵 Omusic"))
	b.WriteString("\n\n")

	// Barra de búsqueda/prompt o estado.
	if m.isInputMode() {
		b.WriteString(m.input.View())
	} else {
		b.WriteString(m.styles.dim.Render(m.status))
	}
	b.WriteString("\n")
	if !l.compactChrome {
		b.WriteString("\n")
	}

	// Sección media: sidebar (nav + cola) y main (portada apilada sobre la
	// letra). Los bloques de enriquecimiento solo se dibujan cuando su
	// servicio está activo; con los toggles apagados la vista es la de la
	// Fase 2.
	b.WriteString(m.renderMiddleSection(l))
	b.WriteString("\n")
	if !l.compactChrome {
		b.WriteString("\n")
	}

	// Tarjeta de "ahora suena" entre el cuerpo y la ayuda (design D3/D5).
	b.WriteString(m.renderNowPlayingCard(l))
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

// renderQueueAt dibuja el panel de cola con las dimensiones del layout,
// envolviendo el cuerpo en la caja de panel histórica (la sidebar de altura
// completa usa el mismo cuerpo via renderSidebar).
func (m Model) renderQueueAt(l layout) string {
	return m.styles.panel.Width(l.queueW).Render(m.queueBody(l))
}

// queueBody compone el contenido del panel de cola: encabezado, ventana
// deslizante y marcadores ▲/▼. Una cola más larga que l.maxQueueRows (p.ej.
// una playlist importada) se muestra como una ventana alrededor de la pista
// actual, evitando que el panel crezca sin límite y rompa el layout. La cola
// interna se mantiene completa. Cada fila descuenta 6 columnas del ancho del
// panel: 2 de padding, 2 de marca de caché y 2 del prefijo ▶/espacios.
func (m Model) queueBody(l layout) string {
	var b strings.Builder
	items := m.queue.Items()
	total := len(items)
	heading := "Cola"
	if total > 0 {
		heading = fmt.Sprintf("Cola (%d)", total)
	}
	b.WriteString(sectionHeader(m.styles, heading))
	b.WriteString("\n")
	if total == 0 {
		b.WriteString(m.styles.dim.Render("(vacía)"))
		return b.String()
	}

	// Los marcadores ▲/▼ solo se dibujan si caben en el interior de la caja
	// (descontando el bloque nav cuando está presente); se omiten antes que
	// filas de pista (fila ▶ visible: Parity @slice1).
	start, end := queueWindow(m.queue.Index(), total, l.maxQueueRows)
	free := l.sidebarH - panelBorder - l.navRows - 1 - (end - start)
	showDown := end < total && free > 0
	if showDown {
		free--
	}
	if start > 0 && free > 0 {
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
	if showDown {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  ▼ %d más", total-end)))
	}
	return strings.TrimSuffix(b.String(), "\n")
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

// renderNowPlaying compone la fila única de "ahora suena" (contenido compacto
// de la tarjeta al pie). El glifo de estado lleva color de estado: turquesa al
// reproducir, apagado en pausa.
func (m Model) renderNowPlaying(l layout) string {
	cur, ok := m.queue.Current()
	if !ok {
		return m.styles.dim.Render("Nada en reproducción")
	}
	bar := progressBar(m.pos, m.dur, l.progressW)
	return fmt.Sprintf("%s %s  %s  %s/%s  vol %d",
		m.stateGlyph(),
		m.styles.current.Render(truncate(cur.Title, l.nowTitleTrunc)),
		bar,
		fmtTime(m.pos), fmtTime(m.dur),
		m.player.Volume(),
	)
}

// stateGlyph devuelve el glifo de estado de reproducción con su color de
// estado: ▶ turquesa mientras suena, ⏸ apagado en pausa.
func (m Model) stateGlyph() string {
	if m.player.Paused() {
		return m.styles.dim.Render("⏸")
	}
	return m.styles.current.Render("▶")
}

// renderNowPlayingCard dibuja la tarjeta de "ahora suena" al pie del cuerpo
// (design D3/D5): caja acentuada translúcida con paridad completa — estado
// ▶/⏸, título, barra de progreso, pos/dur y vol N — en dos filas, o en la
// fila única histórica bajo chrome compacto. Abarca el ancho del cuerpo.
func (m Model) renderNowPlayingCard(l layout) string {
	cardW := l.sidebarW + l.mainW + panelBorder
	rows := 2
	if l.compactChrome {
		rows = 1
	}
	content := m.renderNowPlaying(l)
	if cur, ok := m.queue.Current(); ok && rows == 2 {
		content = fmt.Sprintf("%s %s\n%s  %s/%s  vol %d",
			m.stateGlyph(),
			m.styles.current.Render(truncate(cur.Title, l.nowTitleTrunc)),
			progressBar(m.pos, m.dur, l.progressW),
			fmtTime(m.pos), fmtTime(m.dur),
			m.player.Volume(),
		)
	}
	return m.styles.card.Width(cardW).Height(rows).Render(content)
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

// helpLibraryText es la línea de ayuda del modo biblioteca; se dibuja dentro
// de la caja main envuelta a su ancho interior.
const helpLibraryText = "↑/↓ navegar · n/p sección · enter reproducir · f favorito · a +playlist · c crear playlist · esc/L volver · q salir"

// renderLibraryInMain compone el contenido del modo biblioteca para la caja
// main (design D8): encabezado acentuado, prompt de crear playlist cuando
// aplica, pestañas con la activa entre corchetes, lista con cursor ➤ y su
// ayuda propia. Devuelve el contenido SIN caja: renderMain lo envuelve en la
// caja de altura completa, con la sidebar persistente al lado ("Biblioteca"
// acentuada en la nav). Las filas usan los estilos selected/dim existentes —
// sin fondos opacos.
func (m Model) renderLibraryInMain(l layout) string {
	var b strings.Builder
	b.WriteString(sectionHeader(m.styles, "📚 Biblioteca"))
	b.WriteString("\n")
	if m.mode == modeCreatePlaylist {
		b.WriteString(m.input.View())
		b.WriteString("\n")
	}

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
		lines := playlistLines(m.libPlaylists)
		for i := range lines {
			lines[i] = truncate(lines[i], l.libLineTrunc)
		}
		b.WriteString(m.renderLibList(lines, "(sin playlists)"))
	case sectionFavorites:
		b.WriteString(m.renderLibList(trackLines(m.libFavorites, l.libLineTrunc), "(sin favoritos)"))
	case sectionHistory:
		b.WriteString(m.renderLibList(trackLines(m.libHistory, l.libLineTrunc), "(historial vacío)"))
	}

	b.WriteString("\n")
	// Ayuda de biblioteca envuelta al ancho interior de la caja main para que
	// ninguna línea desborde (el padding de la caja descuenta 2 columnas).
	if maxW := l.mainW - 2; maxW > 0 && lipgloss.Width(helpLibraryText) > maxW {
		b.WriteString(m.styles.help.Width(maxW).Render(helpLibraryText))
	} else {
		b.WriteString(m.styles.help.Render(helpLibraryText))
	}
	return b.String()
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

// renderMiddleSection compone la sección media como dos columnas de altura
// completa: sidebar (cola) y main (enriquecimiento), unidas con JoinHorizontal
// (design D3). Ambas cajas llegan forzadas a bodyH filas desde sus renderers
// (design D6), así que la unión mide exactamente bodyH filas y no queda banda
// en blanco entre el cuerpo y la ayuda.
func (m Model) renderMiddleSection(l layout) string {
	sidebar := m.renderSidebar(l)
	if l.mainW <= 0 {
		// Terminal tan estrecha que no cabe área main: solo la sidebar.
		return sidebar
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, m.renderMain(l))
}

// fillBoxHeight fuerza una caja con borde a ocupar exactamente rows filas en
// total (design D6): Height fija el bloque interior a rows-panelBorder y el
// borde inferior aterriza en la última fila. Height rellena contenido corto
// pero NO encoge contenido más alto que su objetivo, así que un contenido que
// exceda las filas interiores (p.ej. el subpanel de letra con su chrome propio
// cuando bodyH toca el piso) se recorta ANTES de renderizar: la caja nunca
// crece más allá de rows y la vista compuesta no desborda la terminal (design
// D6 / Layout Resilience: recortar en vez de crecer). Si Height no rellenara
// (fallback documentado en el design), PlaceVertical completa el alto
// restante; el assert de banda-en-blanco de las pruebas vigila este camino.
func fillBoxHeight(box lipgloss.Style, w, rows int, content string) string {
	inner := rows - panelBorder
	if lines := strings.Split(content, "\n"); len(lines) > inner {
		content = strings.Join(lines[:inner], "\n")
	}
	out := box.Width(w).Height(inner).Render(content)
	if lipgloss.Height(out) < rows {
		out = lipgloss.PlaceVertical(rows, lipgloss.Top, out)
	}
	return out
}

// sectionHeader compone un encabezado de sección con barra de acento (design
// D7c): un glifo ▎ mauve antecede la etiqueta, estableciendo jerarquía visual
// sin añadir filas ni fondo opaco.
func sectionHeader(s styles, label string) string {
	return s.accentBar.Render("▎") + s.heading.Render(label)
}

// navHeader dibuja el bloque de navegación estático de la sidebar (design
// D4a/D8): el ítem activo en acento — Biblioteca en modo biblioteca, Cola en
// el resto —, los demás apagados, cerrado por una regla acentuada. Sin
// keybindings — jerarquía visual.
func (m Model) navHeader(l layout) string {
	active := "Cola"
	if m.mode == modeLibrary || m.mode == modeCreatePlaylist {
		active = "Biblioteca"
	}
	var b strings.Builder
	for _, item := range []string{"Cola", "Biblioteca", "Favoritos", "Historial"} {
		if item == active {
			b.WriteString(m.styles.navActive.Render("▸ " + item))
		} else {
			b.WriteString(m.styles.navItem.Render("  " + item))
		}
		b.WriteString("\n")
	}
	b.WriteString(m.styles.accentBar.Render(strings.Repeat("─", max(l.sidebarW-2, 1))))
	return b.String()
}

// renderSidebar dibuja la columna lateral de altura completa: una caja de
// ancho sidebarW forzada a sidebarH filas cuyo contenido es el bloque nav
// (cuando cabe) sobre el cuerpo de la cola. En modo slimRail (narrow) el
// contenido es el mismo, ya comprimido por los anchos del layout (títulos
// truncados a queueW-6).
func (m Model) renderSidebar(l layout) string {
	content := m.queueBody(l)
	if l.navRows > 0 {
		content = m.navHeader(l) + "\n" + content
	}
	return fillBoxHeight(m.styles.sidebar, l.sidebarW, l.sidebarH, content)
}

// renderMain dibuja la columna principal de altura completa: una caja de
// ancho mainW forzada a mainH filas con la portada APILADA sobre la letra a
// ancho completo (design D4b/D2f), sin subpaneles con borde propio. Con los
// servicios apagados la caja queda vacía (paridad de elementos: sin
// encabezados fantasma). En modo biblioteca la caja aloja el contenido de la
// biblioteca en lugar de portada+letra (design D8); resultados y pickers NO
// pasan por aquí — siguen a pantalla completa desde View().
func (m Model) renderMain(l layout) string {
	if m.mode == modeLibrary || m.mode == modeCreatePlaylist {
		return fillBoxHeight(m.styles.panel, l.mainW, l.mainH, m.renderLibraryInMain(l))
	}
	return fillBoxHeight(m.styles.panel, l.mainW, l.mainH, m.mainBody(l))
}

// mainBody compone el contenido apilado del área main: bloque de portada
// (cuando su servicio está activo y el breakpoint la muestra) sobre el bloque
// de letra. Devuelve "" cuando ninguno está activo (paridad con la Fase 2).
func (m Model) mainBody(l layout) string {
	hasLyrics := m.lyrics != nil
	hasArtwork := m.artwork != nil && l.showArtwork
	if !hasLyrics && !hasArtwork {
		return ""
	}
	var parts []string
	if hasArtwork {
		parts = append(parts, m.artworkBlock(l))
	}
	if hasLyrics {
		parts = append(parts, m.lyricsBlock(l))
	}
	return strings.Join(parts, "\n")
}

// renderLyricsPanel conserva la firma histórica (la invocan pruebas del
// paquete); deriva el layout del tamaño actual del modelo.
func (m Model) renderLyricsPanel() string {
	return m.renderLyricsPanelAt(computeLayout(m.width, m.height))
}

// renderLyricsPanelAt conserva la variante encajada histórica (la usan las
// pruebas del paquete vía renderLyricsPanel): el bloque de letra dentro de la
// caja de panel.
func (m Model) renderLyricsPanelAt(l layout) string {
	return m.styles.panel.Width(l.lyricsW).Render(m.lyricsBlock(l))
}

// lyricsBlock dibuja el bloque de letra sin caja propia (va apilado dentro de
// la caja main): encabezado acentuado más la ventana sincronizada, el texto
// plano o el estado vacío "sin letra".
func (m Model) lyricsBlock(l layout) string {
	var b strings.Builder
	b.WriteString(sectionHeader(m.styles, "Letra"))
	b.WriteString("\n")

	switch {
	case m.curLyrics.Empty():
		b.WriteString(m.styles.dim.Render("sin letra"))
	case m.curLyrics.Synced:
		b.WriteString(m.renderSyncedLyrics(l))
	default:
		b.WriteString(truncateLines(m.curLyrics.Plain, l.lyricsW-2, l.plainLines))
	}
	return b.String()
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

// renderArtworkPanelAt conserva la variante encajada histórica (la usan las
// pruebas del paquete vía renderArtworkPanel): el bloque de portada dentro de
// la caja de panel.
func (m Model) renderArtworkPanelAt(l layout) string {
	return m.styles.panel.Width(l.artW).Render(m.artworkBlock(l))
}

// artworkBlock dibuja el bloque de portada sin caja propia (va apilado sobre
// la letra dentro de la caja main), o el estado de degradación cuando no hay
// portada. La portada cede filas antes de que la ventana de letra baje de su
// mínimo (design: riesgo de desborde de las 12 filas fijas a alturas cortas).
func (m Model) artworkBlock(l layout) string {
	var b strings.Builder
	b.WriteString(sectionHeader(m.styles, "Portada"))
	b.WriteString("\n")
	if m.curArtwork == "" {
		b.WriteString(m.styles.dim.Render("[sin portada]"))
		return b.String()
	}
	artMax := l.mainH - panelBorder - 1
	if m.lyrics != nil {
		artMax -= 1 + l.plainLines
	}
	b.WriteString(truncateLines(m.curArtwork, l.mainW-2, max(artMax, 1)))
	return b.String()
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
