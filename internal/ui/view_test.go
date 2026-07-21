package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/search"
)

// TestViewGolden captura la salida de View() en dos tamaños de terminal para
// detectar regresiones visuales del rediseño Caelestia.
func TestViewGolden(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{"60x20", 60, 20},
		{"80x24", 80, 24},
		{"120x30", 120, 30},
		{"120x40", 120, 40},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t, Services{
				Lyrics:  fakeLyrics{},
				Artwork: fakeArtwork{art: "ART"},
			})
			m.width, m.height = tc.width, tc.height
			m.queue.Add(search.Result{ID: "a", Title: "Alpha Song", Uploader: "Alpha Artist"})
			m.queue.Add(search.Result{ID: "b", Title: "Beta Track", Uploader: "Beta Artist"})
			m.curTrackID = "a"
			m.curLyrics = lyrics.Lyrics{Plain: "Line one\nLine two\nLine three"}
			m.curArtwork = "ASCII ART"
			m.pos = 45
			m.dur = 180

			out := m.View()
			path := filepath.Join("testdata", "view_"+tc.name+".golden")
			compareGolden(t, path, out)
		})
	}
}

// TestBodyFitsHeight verifica el invariante de Layout Resilience (@slice1): la
// vista compuesta nunca excede el alto de la terminal en los cuatro tamaños de
// los goldens. Guarda en particular el caso 60×20, donde el subpanel de letra
// (con su chrome propio de borde y encabezado) podía hacer crecer la caja main
// por encima de bodyH y desplazar el visualizador fuera de pantalla.
func TestBodyFitsHeight(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{"60x20", 60, 20},
		{"80x24", 80, 24},
		{"120x30", 120, 30},
		{"120x40", 120, 40},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t, Services{
				Lyrics:  fakeLyrics{},
				Artwork: fakeArtwork{art: "ART"},
			})
			m.width, m.height = tc.width, tc.height
			m.queue.Add(search.Result{ID: "a", Title: "Alpha Song", Uploader: "Alpha Artist"})
			m.queue.Add(search.Result{ID: "b", Title: "Beta Track", Uploader: "Beta Artist"})
			m.curTrackID = "a"
			m.curLyrics = lyrics.Lyrics{Plain: "Line one\nLine two\nLine three"}
			m.curArtwork = "ASCII ART"
			m.pos = 45
			m.dur = 180

			if got := lipgloss.Height(m.View()); got > m.height {
				t.Errorf("View() mide %d filas; debe caber en la terminal de %d", got, m.height)
			}
		})
	}
}

// hasNoBackground informa si el estilo no define ningún color de fondo. En
// lipgloss v1 un Background sin definir se lee como NoColor{}; se acepta
// también Color("") como valor vacío equivalente.
func hasNoBackground(s lipgloss.Style) bool {
	switch c := s.GetBackground().(type) {
	case lipgloss.NoColor:
		return true
	case lipgloss.Color:
		return c == lipgloss.Color("")
	default:
		return false
	}
}

// TestStylesNoBackground verifica el escenario "No opaque background on any
// style including new ones": ni los estilos históricos (title/panel) ni los
// cinco del rediseño sidebar (sidebar, card, navActive, navItem, accentBar)
// exponen color de fondo, y las cajas conservan el borde redondeado.
func TestStylesNoBackground(t *testing.T) {
	s := defaultStyles()
	checks := []struct {
		name  string
		style lipgloss.Style
		boxed bool
	}{
		{"title", s.title, true},
		{"panel", s.panel, true},
		{"sidebar", s.sidebar, true},
		{"card", s.card, true},
		{"navActive", s.navActive, false},
		{"navItem", s.navItem, false},
		{"accentBar", s.accentBar, false},
	}
	for _, c := range checks {
		if !hasNoBackground(c.style) {
			t.Errorf("styles.%s no debe definir Background; got %#v", c.name, c.style.GetBackground())
		}
		if c.boxed && c.style.GetBorderStyle() != lipgloss.RoundedBorder() {
			t.Errorf("styles.%s debe conservar el borde redondeado", c.name)
		}
	}
}

// TestClassifyBoundaries verifica los valores de frontera exactos de los
// breakpoints del design: narrow < 90 ≤ medium < 120 ≤ wide.
func TestClassifyBoundaries(t *testing.T) {
	cases := []struct {
		width int
		want  breakpoint
	}{
		{59, bpNarrow},
		{60, bpNarrow},
		{89, bpNarrow},
		{90, bpMedium},
		{119, bpMedium},
		{120, bpWide},
	}
	for _, tc := range cases {
		if got := classify(tc.width); got != tc.want {
			t.Errorf("classify(%d) = %v; want %v", tc.width, got, tc.want)
		}
	}
}

// TestComputeLayoutWidths verifica en las fronteras de breakpoint la división
// principal sidebar | main (design D1): el invariante de suma exacta D1d, los
// mínimos de rail/sidebar, que main nunca colapsa y que slimRail y la portada
// siguen al breakpoint narrow.
func TestComputeLayoutWidths(t *testing.T) {
	const railMin, sbMin = 16, 26
	for _, width := range []int{59, 60, 89, 90, 119, 120} {
		t.Run(fmt.Sprintf("%dcols", width), func(t *testing.T) {
			l := computeLayout(width, 24)
			usable := max(width-2, minUsable)
			if got := l.sidebarW + l.mainW + 2*panelBorder; got != usable {
				t.Errorf("sidebarW+mainW+2*panelBorder = %d; want usable %d", got, usable)
			}
			if l.mainW <= 0 {
				t.Errorf("mainW %d debe ser positivo", l.mainW)
			}
			if wantRail := l.bp == bpNarrow; l.slimRail != wantRail {
				t.Errorf("slimRail = %v; want %v (bp=%v)", l.slimRail, wantRail, l.bp)
			}
			if l.slimRail {
				if l.sidebarW < railMin {
					t.Errorf("sidebarW %d < mínimo de rail %d", l.sidebarW, railMin)
				}
			} else if l.sidebarW < sbMin {
				t.Errorf("sidebarW %d < mínimo %d", l.sidebarW, sbMin)
			}
			if l.bp == bpNarrow {
				if l.artW != 0 || l.showArtwork {
					t.Errorf("narrow debe ocultar portada: artW=%d showArtwork=%v", l.artW, l.showArtwork)
				}
			} else if l.artW <= 0 || !l.showArtwork {
				t.Errorf("medium/wide deben mostrar portada: artW=%d showArtwork=%v", l.artW, l.showArtwork)
			}
		})
	}
}

// TestComputeLayoutHeight verifica que las alturas de columna igualan bodyH
// (design D2a) y que las ventanas de cola y letra derivan del alto sin techos
// fijos (design D2b–D2f): mínimos que no colapsan, ventana impar centrada,
// crecimiento monótono con la altura y la aritmética re-medida del slice 2
// (chromeFixed=14 con helpRows(120)=2 ⇒ bodyH = height-16, con chrome
// compacto a alturas mínimas).
func TestComputeLayoutHeight(t *testing.T) {
	prev := 0
	for _, height := range []int{20, 24, 30, 40} {
		t.Run(fmt.Sprintf("%drows", height), func(t *testing.T) {
			l := computeLayout(120, height)
			if l.bodyH < 4 {
				t.Errorf("bodyH %d < mínimo 4", l.bodyH)
			}
			if l.sidebarH != l.bodyH || l.mainH != l.bodyH {
				t.Errorf("sidebarH/mainH = %d/%d; ambos deben igualar bodyH %d",
					l.sidebarH, l.mainH, l.bodyH)
			}
			if l.maxQueueRows < 3 {
				t.Errorf("maxQueueRows %d < mínimo 3", l.maxQueueRows)
			}
			if l.lyricWindow < 3 {
				t.Errorf("lyricWindow %d < mínimo 3", l.lyricWindow)
			}
			if l.lyricWindow%2 != 1 {
				t.Errorf("lyricWindow %d debe ser impar", l.lyricWindow)
			}
			if l.plainLines < 3 {
				t.Errorf("plainLines %d < mínimo 3", l.plainLines)
			}
			if l.maxQueueRows < prev {
				t.Errorf("maxQueueRows %d no debe decrecer al crecer la altura (previo %d)",
					l.maxQueueRows, prev)
			}
			prev = l.maxQueueRows
			switch height {
			case 20:
				// Chrome compacto: 20 - (11+2) = 7 filas de cuerpo.
				if !l.compactChrome {
					t.Error("a 20 filas debe activarse el chrome compacto")
				}
				if l.bodyH != 7 {
					t.Errorf("bodyH = %d; want 7 (20 - (chromeCompact 11 + ayuda 2))", l.bodyH)
				}
				if l.maxQueueRows >= 10 {
					t.Errorf("a 20 filas la cola debe reducirse: maxQueueRows=%d", l.maxQueueRows)
				}
				if l.navRows != 0 {
					t.Errorf("a 20 filas la nav debe ceder ante la cola: navRows=%d", l.navRows)
				}
			case 30:
				// Aritmética D5a: bodyH = 30 - (chromeFixed 14 + ayuda 2) = 14.
				if l.compactChrome {
					t.Error("a 30 filas no debe activarse el chrome compacto")
				}
				if l.bodyH != 14 {
					t.Errorf("bodyH = %d; want 14 (30 - (chromeFixed 14 + ayuda 2))", l.bodyH)
				}
				if l.navRows != 5 {
					t.Errorf("a 30 filas la nav debe dibujarse: navRows=%d", l.navRows)
				}
			case 40:
				// Aritmética D5a: bodyH = 40 - (14+2) = 24; sin techo fijo la
				// cola llena sidebarH-queueChrome(10) = 14 filas y crece más
				// que la ventana de letra.
				if l.bodyH != 24 {
					t.Errorf("bodyH = %d; want 24 (40 - (chromeFixed 14 + ayuda 2))", l.bodyH)
				}
				if l.maxQueueRows != l.sidebarH-10 {
					t.Errorf("maxQueueRows = %d; debe derivar de sidebarH-queueChrome = %d",
						l.maxQueueRows, l.sidebarH-10)
				}
				if l.maxQueueRows <= l.lyricWindow {
					t.Errorf("a 40 filas la cola debe crecer más que la letra: maxQueueRows=%d lyricWindow=%d",
						l.maxQueueRows, l.lyricWindow)
				}
				if l30 := computeLayout(120, 30); l.lyricWindow <= l30.lyricWindow {
					t.Errorf("la ventana de letra debe crecer de 30 a 40 filas: %d <= %d",
						l.lyricWindow, l30.lyricWindow)
				}
			}
		})
	}
}

// TestQueueCurrentVisibleLongQueue60x20 verifica Element Parity (@slice1) con
// cola larga: a 60×20 con 30 pistas y la actual en el medio, la fila ▶ actual
// sobrevive (marcadores ▲/▼ omitidos antes que filas) y quedan ≥3 filas.
func TestQueueCurrentVisibleLongQueue60x20(t *testing.T) {
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}, Artwork: fakeArtwork{art: "ART"}})
	m.width, m.height = 60, 20
	for i := 0; i < 30; i++ {
		m.queue.Add(search.Result{ID: fmt.Sprintf("t%02d", i), Title: fmt.Sprintf("Track %02d", i)})
	}
	for i := 0; i < 15; i++ {
		m.queue.Next()
	}
	sidebar := m.renderSidebar(computeLayout(m.width, m.height))
	if !strings.Contains(sidebar, "▶ Track 15") {
		t.Errorf("la fila ▶ de la pista actual (Track 15) debe verse en la sidebar:\n%s", sidebar)
	}
	if rows := strings.Count(sidebar, "Track "); rows < 3 {
		t.Errorf("la ventana de cola debe conservar ≥3 filas de pista; got %d:\n%s", rows, sidebar)
	}
}

// Test60x20NarrowNoArtwork verifica el escenario "Narrow breakpoint hides
// artwork": a 60×20, con servicios de letra y portada activos, la portada no
// se dibuja y cola + letra siguen presentes.
func Test60x20NarrowNoArtwork(t *testing.T) {
	m := newTestModel(t, Services{
		Lyrics:  fakeLyrics{},
		Artwork: fakeArtwork{art: "ART"},
	})
	m.width, m.height = 60, 20
	m.queue.Add(search.Result{ID: "a", Title: "Alpha Song"})
	m.curTrackID = "a"
	m.curLyrics = lyrics.Lyrics{Plain: "Line one\nLine two"}
	m.curArtwork = "ASCII ART"

	out := m.View()
	if strings.Contains(out, "Portada") || strings.Contains(out, "ASCII ART") {
		t.Errorf("narrow no debe mostrar el panel de portada; got:\n%s", out)
	}
	if !strings.Contains(out, "Cola") {
		t.Errorf("la cola debe seguir presente en narrow; got:\n%s", out)
	}
	if !strings.Contains(out, "Letra") {
		t.Errorf("la letra debe seguir presente en narrow; got:\n%s", out)
	}
}

// TestNoLineExceedsWidth verifica el escenario "No rendered line exceeds
// terminal width" en 60, 80 y 120 columnas (incluida la combinación 60×20 del
// breakpoint narrow), con títulos y letra largos para forzar los truncados.
func TestNoLineExceedsWidth(t *testing.T) {
	longTitle := strings.Repeat("Título Muy Largo ", 8)
	sizes := []struct{ width, height int }{
		{60, 20},
		{60, 24},
		{80, 24},
		{120, 24},
		{120, 30},
		{120, 40},
	}
	for _, size := range sizes {
		width := size.width
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := newTestModel(t, Services{
				Lyrics:  fakeLyrics{},
				Artwork: fakeArtwork{art: "ART"},
			})
			m.width, m.height = size.width, size.height
			m.queue.Add(search.Result{ID: "a", Title: longTitle, Uploader: "Artista"})
			for i := 0; i < 20; i++ {
				m.queue.Add(search.Result{ID: fmt.Sprintf("v%02d", i), Title: longTitle})
			}
			m.curTrackID = "a"
			m.curLyrics = lyrics.Lyrics{Synced: true, Lines: []lyrics.Line{
				{T: 0, Text: strings.Repeat("verso largo ", 10)},
				{T: 10, Text: strings.Repeat("otro verso ", 10)},
				{T: 20, Text: "corto"},
			}}
			m.curArtwork = "ASCII ART"
			m.pos, m.dur = 45, 180

			out := m.View()
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("línea %d excede el ancho: %d > %d\n%q", i, w, width, line)
				}
			}
		})
	}
}

// TestGoldensDiffer verifica los escenarios "80×24 and 120×30 goldens differ"
// y "Breakpoints render distinct deterministic layouts": los cuatro fixtures
// responsivos deben diferir entre sí por pares (el 120×40 añade la variación
// solo-en-alto del mismo breakpoint wide).
func TestGoldensDiffer(t *testing.T) {
	names := []string{"view_60x20.golden", "view_80x24.golden", "view_120x30.golden", "view_120x40.golden"}
	goldens := make([][]byte, len(names))
	for i, name := range names {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Skipf("golden ausente; regenera con UPDATE_GOLDEN=1 (%v)", err)
		}
		goldens[i] = data
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if bytes.Equal(goldens[i], goldens[j]) {
				t.Errorf("%s y %s no deben ser byte-idénticos", names[i], names[j])
			}
		}
	}
}

// TestNoBlankBodyBand verifica el escenario "No blank vertical band at
// 120x40": las dos columnas del cuerpo (sidebar y main) llegan forzadas a
// bodyH filas, así que ninguna fila del cuerpo puede quedar totalmente en
// blanco entre el chrome superior y la ayuda (toda fila contiene al menos un
// glifo de borde o contenido).
func TestNoBlankBodyBand(t *testing.T) {
	m := newTestModel(t, Services{
		Lyrics:  fakeLyrics{},
		Artwork: fakeArtwork{art: "ART"},
	})
	m.width, m.height = 120, 40
	m.queue.Add(search.Result{ID: "a", Title: "Alpha Song", Uploader: "Alpha Artist"})
	m.curTrackID = "a"
	m.curArtwork = "ASCII ART"
	out := m.View()
	lines := strings.Split(out, "\n")
	// Chrome superior: título (3), separador, estado, separador = 6 filas
	// antes del cuerpo. Chrome inferior: separador, tarjeta (4), separador,
	// ayuda envuelta (2 a 120 cols), visualizador y línea final = 10 filas.
	bodyStart, bodyEnd := 6, len(lines)-10
	if bodyEnd <= bodyStart {
		t.Fatalf("salida demasiado corta para aislar el cuerpo: %d líneas", len(lines))
	}
	for i := bodyStart; i < bodyEnd && i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			t.Errorf("fila %d del cuerpo totalmente en blanco: banda vacía en la sección media", i)
		}
	}
}

// TestFooterCardNoClip60x20 verifica el escenario "Footer card does not clip
// elements at 20 rows": a 60×20 con la tarjeta al pie presente, título,
// tarjeta (estado + vol), encabezado de cola, ayuda y visualizador siguen
// visibles, ninguna línea excede 60 columnas y la vista compuesta cabe en la
// terminal.
func TestFooterCardNoClip60x20(t *testing.T) {
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}, Artwork: fakeArtwork{art: "ART"}})
	m.width, m.height = 60, 20
	m.queue.Add(search.Result{ID: "a", Title: "Alpha Song", Uploader: "Alpha Artist"})
	m.curTrackID = "a"
	m.pos, m.dur = 45, 180

	out := m.View()
	if !strings.Contains(out, "Omusic") {
		t.Errorf("el título debe seguir visible a 60×20; got:\n%s", out)
	}
	if !strings.Contains(out, "▶") && !strings.Contains(out, "⏸") {
		t.Errorf("la tarjeta debe mostrar el glifo de estado ▶/⏸; got:\n%s", out)
	}
	if !strings.Contains(out, "vol ") {
		t.Errorf("la tarjeta debe mostrar el volumen; got:\n%s", out)
	}
	if !strings.Contains(out, "0:45/3:00") {
		t.Errorf("la tarjeta debe mostrar el tiempo pos/dur; got:\n%s", out)
	}
	if !strings.Contains(out, "Cola") {
		t.Errorf("el encabezado de cola debe seguir visible; got:\n%s", out)
	}
	if !strings.Contains(out, "buscar") {
		t.Errorf("la ayuda debe seguir visible; got:\n%s", out)
	}
	if !strings.ContainsAny(out, "▁▂▃▄▅▆▇█") {
		t.Errorf("el visualizador debe seguir visible; got:\n%s", out)
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Errorf("línea %d excede 60 columnas: %d\n%q", i, w, line)
		}
	}
	if got := lipgloss.Height(out); got > 20 {
		t.Errorf("la vista con tarjeta mide %d filas; debe caber en 20", got)
	}
}

// TestFooterCardParity120x30 verifica el escenario "Footer card shows
// now-playing content": la tarjeta al pie conserva paridad completa con la
// barra histórica (glifo de estado, título, barra de progreso, tiempo pos/dur
// y vol N) y el bloque nav de la sidebar dibuja los cuatro ítems estáticos.
func TestFooterCardParity120x30(t *testing.T) {
	m := newTestModel(t, Services{Lyrics: fakeLyrics{}, Artwork: fakeArtwork{art: "ART"}})
	m.width, m.height = 120, 30
	m.queue.Add(search.Result{ID: "a", Title: "Alpha Song", Uploader: "Alpha Artist"})
	m.curTrackID = "a"
	m.pos, m.dur = 45, 180

	out := m.View()
	for _, want := range []string{"▶", "Alpha Song", "━", "─", "0:45/3:00", "vol 70"} {
		if !strings.Contains(out, want) {
			t.Errorf("la tarjeta debe conservar %q; got:\n%s", want, out)
		}
	}
	// Nav estática de la sidebar (design D4a): Cola activa + tres ítems más.
	for _, item := range []string{"Cola", "Biblioteca", "Favoritos", "Historial"} {
		if !strings.Contains(out, item) {
			t.Errorf("la nav de la sidebar debe mostrar %q; got:\n%s", item, out)
		}
	}
	// La tarjeta vive DEBAJO del cuerpo: su borde superior aparece después de
	// la última fila de la sidebar.
	if lipgloss.Height(out) > 30 {
		t.Errorf("la vista mide %d filas; debe caber en 30", lipgloss.Height(out))
	}
}

// TestCaelestiaAccentColors verifica el escenario "All colors match Caelestia
// palette": afirma los colores hexadecimales de la paleta por nombre sobre los
// estilos de defaultStyles(), independiente de los goldens (cierra Obs-1).
func TestCaelestiaAccentColors(t *testing.T) {
	s := defaultStyles()
	cases := []struct {
		name  string
		color lipgloss.Color
	}{
		{"accent mauve (heading/border/viz/errorMsg/selected-border)", "#e0aaff"},
		{"highlight teal (selected/current)", "#00f5d4"},
		{"muted (dim/help)", "#a0a0a0"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			switch tc.color {
			case "#e0aaff":
				if s.heading.GetForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("heading foreground no es mauve: %v", s.heading.GetForeground())
				}
				if s.title.GetForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("title foreground no es mauve: %v", s.title.GetForeground())
				}
				if s.viz.GetForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("viz foreground no es mauve: %v", s.viz.GetForeground())
				}
				if s.errorMsg.GetForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("errorMsg foreground no es mauve: %v", s.errorMsg.GetForeground())
				}
				if s.navActive.GetForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("navActive foreground no es mauve: %v", s.navActive.GetForeground())
				}
				if s.accentBar.GetForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("accentBar foreground no es mauve: %v", s.accentBar.GetForeground())
				}
				if s.sidebar.GetBorderTopForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("sidebar border no es mauve: %v", s.sidebar.GetBorderTopForeground())
				}
				if s.card.GetBorderTopForeground() != lipgloss.Color("#e0aaff") {
					t.Errorf("card border no es mauve: %v", s.card.GetBorderTopForeground())
				}
			case "#00f5d4":
				if s.selected.GetForeground() != lipgloss.Color("#00f5d4") {
					t.Errorf("selected foreground no es teal: %v", s.selected.GetForeground())
				}
				if s.current.GetForeground() != lipgloss.Color("#00f5d4") {
					t.Errorf("current foreground no es teal: %v", s.current.GetForeground())
				}
			case "#a0a0a0":
				if s.dim.GetForeground() != lipgloss.Color("#a0a0a0") {
					t.Errorf("dim foreground no es muted: %v", s.dim.GetForeground())
				}
				if s.help.GetForeground() != lipgloss.Color("#a0a0a0") {
					t.Errorf("help foreground no es muted: %v", s.help.GetForeground())
				}
				if s.navItem.GetForeground() != lipgloss.Color("#a0a0a0") {
					t.Errorf("navItem foreground no es muted: %v", s.navItem.GetForeground())
				}
			}
		})
	}
}

// TestDelegateNoBackground extiende el assert de translucidez al delegate de
// los modales (escenario "Modals, library, and pickers preserved and
// translucent"): ningún subestilo del delegate ni el título tematizado del
// list pueden definir un Background opaco.
func TestDelegateNoBackground(t *testing.T) {
	d := caelestiaListDelegate()
	checks := []struct {
		name  string
		style lipgloss.Style
	}{
		{"NormalTitle", d.Styles.NormalTitle},
		{"NormalDesc", d.Styles.NormalDesc},
		{"SelectedTitle", d.Styles.SelectedTitle},
		{"SelectedDesc", d.Styles.SelectedDesc},
		{"DimmedTitle", d.Styles.DimmedTitle},
		{"DimmedDesc", d.Styles.DimmedDesc},
	}
	for _, c := range checks {
		c := c
		t.Run(c.name, func(t *testing.T) {
			if !hasNoBackground(c.style) {
				t.Errorf("delegate.%s no debe definir Background; got %#v",
					c.name, c.style.GetBackground())
			}
		})
	}
	// La barra de título tematizada tampoco: themedList debe reemplazar el
	// Background("62") que el DefaultStyles de bubbles/list trae por defecto.
	themed := themedList(list.New(nil, list.NewDefaultDelegate(), 0, 0))
	if !hasNoBackground(themed.Styles.Title) {
		t.Errorf("list.Styles.Title tematizado no debe definir Background; got %#v",
			themed.Styles.Title.GetBackground())
	}
	if themed.Styles.Title.GetForeground() != lipgloss.Color("#e0aaff") {
		t.Errorf("list.Styles.Title debe usar foreground mauve; got %v",
			themed.Styles.Title.GetForeground())
	}
}

// TestLibraryViewIsTranslucent verifica la rama library del escenario
// "Modals, library, and pickers preserved and translucent": los ítems y el
// cursor ➤ están presentes y los estilos de selección no definen Background
// (la selección se distingue por color/negrita/prefijo, no por relleno).
func TestLibraryViewIsTranslucent(t *testing.T) {
	m := newTestModel(t, Services{})
	m.mode = modeLibrary
	m.libSection = sectionFavorites
	m.libFavorites = []search.Result{
		{ID: "a", Title: "Canción A", Uploader: "Artista A"},
		{ID: "b", Title: "Canción B", Uploader: "Artista B"},
	}
	m.libCursor = 0
	out := m.View()
	if !strings.Contains(out, "Canción A") {
		t.Errorf("biblioteca debe mostrar los ítems; got:\n%s", out)
	}
	if !strings.Contains(out, "➤") {
		t.Errorf("biblioteca debe mostrar el cursor ➤; got:\n%s", out)
	}
	// Verificar que los estilos de selección no tienen Background.
	if !hasNoBackground(m.styles.selected) {
		t.Errorf("styles.selected no debe definir Background")
	}
	if !hasNoBackground(m.styles.dim) {
		t.Errorf("styles.dim no debe definir Background")
	}
}

// TestResultsModalGolden bloquea el render del modal de resultados a 120×30
// con el delegate Caelestia, para detectar regresiones visuales del modal.
func TestResultsModalGolden(t *testing.T) {
	m := newTestModel(t, Services{})
	m.mode = modeResults
	m.width, m.height = 120, 30
	// Mismo dimensionado que aplica Update ante tea.WindowSizeMsg (alto - 4).
	m.resultsList.SetSize(120, 26)
	items := []list.Item{
		resultItem{r: search.Result{ID: "a", Title: "Canción A", Uploader: "Artista A"}},
		resultItem{r: search.Result{ID: "b", Title: "Canción B", Uploader: "Artista B"}},
		resultItem{r: search.Result{ID: "c", Title: "Canción C", Uploader: "Artista C"}},
		resultItem{r: search.Result{ID: "d", Title: "Canción D", Uploader: "Artista D"}},
		resultItem{r: search.Result{ID: "e", Title: "Canción E", Uploader: "Artista E"}},
	}
	m.resultsList.SetItems(items)
	// themedList explícito para ejercer el mismo path que View() en modeResults.
	m.resultsList = themedList(m.resultsList)
	out := m.View()
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 120 {
			t.Errorf("línea %d del modal excede 120 columnas: %d\n%q", i, w, line)
		}
	}
	compareGolden(t, filepath.Join("testdata", "view_results_120x30.golden"), out)
}

// compareGolden lee un archivo golden y lo compara con el valor actual. Si la
// variable UPDATE_GOLDEN está activa, sobreescribe el golden con el valor actual.
func compareGolden(t *testing.T, path, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("crear directorio golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatalf("escribir golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file no encontrado (%s); ejecuta con UPDATE_GOLDEN=1 para crearlo: %v", path, err)
	}

	if string(want) != got {
		gotPath := path + ".got"
		_ = os.WriteFile(gotPath, []byte(got), 0644)
		t.Fatalf("golden mismatch en %s\nescrito resultado actual en %s\n%s", path, gotPath, diffText(string(want), got))
	}
}

// diffText devuelve un resumen breve de las diferencias entre dos cadenas,
// mostrando las primeras líneas que difieren.
func diffText(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	max := len(wantLines)
	if len(gotLines) > max {
		max = len(gotLines)
	}
	var b strings.Builder
	b.WriteString("primeras líneas diferentes:\n")
	for i := 0; i < max && i < 20; i++ {
		w, g := "", ""
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			b.WriteString("--- " + w + "\n")
			b.WriteString("+++ " + g + "\n")
		}
	}
	return b.String()
}
