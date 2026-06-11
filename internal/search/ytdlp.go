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
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Uploader string  `json:"uploader"`
	Channel  string  `json:"channel"`
	Duration float64 `json:"duration"`
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

	args := []string{
		fmt.Sprintf("ytsearch%d:%s", n, q),
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
		"--ignore-errors",
	}
	cmd := exec.CommandContext(ctx, y.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return parseEntries(stdout.Bytes())
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
