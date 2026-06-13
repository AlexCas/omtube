package search

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseEntries(t *testing.T) {
	data := []byte(`{"id":"abc","title":"Numb","uploader":"Linkin Park","duration":185.0}
{"id":"def","title":"In The End","channel":"LP Channel","duration":216}

{"title":"sin id","duration":100}
{malformed json}
{"id":"ghi","title":"Faint","duration":162}`)

	res, err := parseEntries(data)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("se esperaban 3 resultados, got %d: %+v", len(res), res)
	}
	if res[0].ID != "abc" || res[0].Title != "Numb" || res[0].Uploader != "Linkin Park" || res[0].Duration != 185 {
		t.Fatalf("primer resultado mal parseado: %+v", res[0])
	}
	// uploader vacío usa channel como fallback.
	if res[1].Uploader != "LP Channel" {
		t.Fatalf("fallback a channel falló: %+v", res[1])
	}
}

func TestResultURL(t *testing.T) {
	r := Result{ID: "xyz"}
	if got := r.URL(); got != "https://www.youtube.com/watch?v=xyz" {
		t.Fatalf("URL = %s", got)
	}
}

func TestParsePlaylist(t *testing.T) {
	data := []byte(`{"id":"abc","title":"Numb","uploader":"Linkin Park","playlist_title":"Mix LP"}
{"title":"sin id"}
{"id":"def","title":"Faint","channel":"LP Channel"}`)

	res, title, err := parsePlaylist(data)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if title != "Mix LP" {
		t.Fatalf("title = %q; want \"Mix LP\"", title)
	}
	if len(res) != 2 {
		t.Fatalf("se esperaban 2 pistas (entrada sin id descartada), got %d: %+v", len(res), res)
	}
	if res[0].ID != "abc" || res[1].ID != "def" || res[1].Uploader != "LP Channel" {
		t.Fatalf("pistas mal parseadas: %+v", res)
	}
}

// fakeYtDlp escribe un script ejecutable que ignora sus argumentos e imprime el
// NDJSON indicado en stdout, simulando yt-dlp. Se omite en Windows (usa shell).
func fakeYtDlp(t *testing.T, ndjson string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("script de shell no soportado en Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-ytdlp")
	script := "#!/bin/sh\ncat <<'EOF'\n" + ndjson + "\nEOF\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("escribir fake yt-dlp: %v", err)
	}
	return path
}

func TestResolve(t *testing.T) {
	bin := fakeYtDlp(t, `{"id":"abc123","title":"Numb","uploader":"Linkin Park","duration":185}`)
	y := NewYtDlp(bin)

	r, err := y.Resolve(context.Background(), "https://www.youtube.com/watch?v=abc123&list=PL999")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if r.ID != "abc123" || r.Title != "Numb" || r.Uploader != "Linkin Park" || r.Duration != 185 {
		t.Fatalf("resultado mal resuelto: %+v", r)
	}
}

func TestResolveNoResult(t *testing.T) {
	bin := fakeYtDlp(t, `{"title":"sin id"}`)
	y := NewYtDlp(bin)
	if _, err := y.Resolve(context.Background(), "https://youtu.be/abc123"); err == nil {
		t.Fatal("se esperaba error cuando no se resuelve ningún vídeo")
	}
}

func TestResolvePlaylist(t *testing.T) {
	bin := fakeYtDlp(t, `{"id":"abc","title":"Numb","playlist_title":"Mix LP"}
{"id":"def","title":"Faint"}`)
	y := NewYtDlp(bin)

	tracks, title, err := y.ResolvePlaylist(context.Background(), "https://www.youtube.com/playlist?list=PL999")
	if err != nil {
		t.Fatalf("ResolvePlaylist error: %v", err)
	}
	if title != "Mix LP" {
		t.Fatalf("title = %q", title)
	}
	if len(tracks) != 2 || tracks[0].ID != "abc" || tracks[1].ID != "def" {
		t.Fatalf("pistas mal resueltas: %+v", tracks)
	}
}
