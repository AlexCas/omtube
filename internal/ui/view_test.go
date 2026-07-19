package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
