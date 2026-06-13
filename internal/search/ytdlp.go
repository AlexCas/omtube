package search

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// YtDlp implementa Searcher usando el binario yt-dlp.
type YtDlp struct {
	// Bin es la ruta al ejecutable yt-dlp (por defecto "yt-dlp").
	Bin string
}

// NewYtDlp crea un buscador con el binario indicado.
func NewYtDlp(bin string) *YtDlp {
	if bin == "" {
		bin = "yt-dlp"
	}
	return &YtDlp{Bin: bin}
}

// ytdlpEntry refleja los campos de cada línea JSON emitida por yt-dlp.
type ytdlpEntry struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Uploader      string  `json:"uploader"`
	Channel       string  `json:"channel"`
	Duration      float64 `json:"duration"`
	PlaylistTitle string  `json:"playlist_title"`
	Playlist      string  `json:"playlist"`
}

// run ejecuta yt-dlp con los argumentos dados y devuelve su stdout. Centraliza el
// manejo de error para Search/Resolve/ResolvePlaylist.
func (y *YtDlp) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, y.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// Search ejecuta `yt-dlp "ytsearchN:q" --dump-json --flat-playlist` y parsea la
// salida NDJSON. Las entradas sin id se descartan.
func (y *YtDlp) Search(ctx context.Context, q string, n int) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if n <= 0 {
		n = 10
	}

	out, err := y.run(ctx,
		fmt.Sprintf("ytsearch%d:%s", n, q),
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
		"--ignore-errors",
	)
	if err != nil {
		return nil, err
	}
	return parseEntries(out)
}

// Resolve resuelve una URL de vídeo de YouTube a un único resultado con su
// metadato completo (título, autor, duración). Usa `--no-playlist` para ignorar
// cualquier parámetro `list=` y resolver solo el vídeo. Si la URL clasifica como
// vídeo, se reconstruye su URL canónica para evitar arrastrar parámetros extra.
func (y *YtDlp) Resolve(ctx context.Context, rawURL string) (Result, error) {
	target := strings.TrimSpace(rawURL)
	if target == "" {
		return Result{}, fmt.Errorf("URL vacía")
	}
	if kind, id := ClassifyURL(target); kind == URLVideo {
		target = videoURL(id)
	}
	out, err := y.run(ctx, target,
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
	)
	if err != nil {
		return Result{}, err
	}
	results, err := parseEntries(out)
	if err != nil {
		return Result{}, err
	}
	if len(results) == 0 {
		return Result{}, fmt.Errorf("yt-dlp: la URL no resolvió ningún vídeo")
	}
	return results[0], nil
}

// ResolvePlaylist resuelve una URL de playlist de YouTube a sus entradas en orden
// (cada una con id, título y autor) junto con el título de la playlist. Usa
// `--flat-playlist` para no descargar audio. Las entradas sin id se descartan.
func (y *YtDlp) ResolvePlaylist(ctx context.Context, rawURL string) ([]Result, string, error) {
	target := strings.TrimSpace(rawURL)
	if target == "" {
		return nil, "", fmt.Errorf("URL vacía")
	}
	if kind, id := ClassifyURL(target); kind == URLPlaylist {
		target = playlistURL(id)
	}
	out, err := y.run(ctx, target,
		"--dump-json",
		"--flat-playlist",
		"--yes-playlist",
		"--no-warnings",
		"--ignore-errors",
	)
	if err != nil {
		return nil, "", err
	}
	return parsePlaylist(out)
}

// parseEntries convierte NDJSON de yt-dlp en resultados, descartando entradas sin id.
func parseEntries(data []byte) ([]Result, error) {
	var out []Result
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e ytdlpEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // línea malformada: se ignora
		}
		if e.ID == "" {
			continue
		}
		uploader := e.Uploader
		if uploader == "" {
			uploader = e.Channel
		}
		out = append(out, Result{
			ID:       e.ID,
			Title:    e.Title,
			Uploader: uploader,
			Duration: int(e.Duration),
		})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// parsePlaylist convierte NDJSON de una playlist en resultados (descartando
// entradas sin id) y extrae el título de la playlist de la primera entrada que lo
// declare.
func parsePlaylist(data []byte) ([]Result, string, error) {
	var (
		out   []Result
		title string
	)
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e ytdlpEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // línea malformada: se ignora
		}
		if title == "" {
			if e.PlaylistTitle != "" {
				title = e.PlaylistTitle
			} else if e.Playlist != "" {
				title = e.Playlist
			}
		}
		if e.ID == "" {
			continue
		}
		uploader := e.Uploader
		if uploader == "" {
			uploader = e.Channel
		}
		out = append(out, Result{
			ID:       e.ID,
			Title:    e.Title,
			Uploader: uploader,
			Duration: int(e.Duration),
		})
	}
	if err := sc.Err(); err != nil {
		return nil, "", err
	}
	return out, title, nil
}
