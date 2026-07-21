package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{"80x24", 80, 24},
		{"120x30", 120, 30},
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

// TestStylesNoBackground verifica el escenario "No opaque background paints
// over the terminal glass": ni title ni panel exponen color de fondo y ambos
// conservan el borde redondeado.
func TestStylesNoBackground(t *testing.T) {
	s := defaultStyles()
	if !hasNoBackground(s.title) {
		t.Errorf("styles.title no debe definir Background; got %#v", s.title.GetBackground())
	}
	if !hasNoBackground(s.panel) {
		t.Errorf("styles.panel no debe definir Background; got %#v", s.panel.GetBackground())
	}
	if s.title.GetBorderStyle() != lipgloss.RoundedBorder() {
		t.Error("styles.title debe conservar el borde redondeado")
	}
	if s.panel.GetBorderStyle() != lipgloss.RoundedBorder() {
		t.Error("styles.panel debe conservar el borde redondeado")
	}
}

// TestNoLineExceedsWidth verifica el escenario "No rendered line exceeds
// terminal width" en 60, 80 y 120 columnas, con títulos y letra largos para
// forzar los truncados fluidos.
func TestNoLineExceedsWidth(t *testing.T) {
	longTitle := strings.Repeat("Título Muy Largo ", 8)
	for _, width := range []int{60, 80, 120} {
		t.Run(fmt.Sprintf("%dcols", width), func(t *testing.T) {
			m := newTestModel(t, Services{
				Lyrics:  fakeLyrics{},
				Artwork: fakeArtwork{art: "ART"},
			})
			m.width, m.height = width, 24
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

// TestGoldensDiffer verifica el escenario "80×24 and 120×30 goldens differ":
// tras el rediseño responsivo ambos fixtures no pueden ser idénticos.
func TestGoldensDiffer(t *testing.T) {
	want80, err80 := os.ReadFile(filepath.Join("testdata", "view_80x24.golden"))
	want120, err120 := os.ReadFile(filepath.Join("testdata", "view_120x30.golden"))
	if err80 != nil || err120 != nil {
		t.Skipf("goldens ausentes; regenera con UPDATE_GOLDEN=1 (%v, %v)", err80, err120)
	}
	if bytes.Equal(want80, want120) {
		t.Fatal("view_80x24.golden y view_120x30.golden no deben ser byte-idénticos")
	}
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
