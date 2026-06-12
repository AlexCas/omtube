package cache

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// Service descarga audio mediante yt-dlp al directorio de caché y mantiene el
// índice en la base de datos compartida. Cada repetición de una pista cacheada
// se sirve desde el archivo local. La caché expira por tamaño y antigüedad.
type Service struct {
	idx      *Index
	ytdlp    string        // ruta al binario yt-dlp
	dir      string        // directorio raíz de caché (XDG)
	maxBytes int64         // límite de tamaño total; <=0 desactiva el límite
	maxAge   time.Duration // antigüedad máxima por entrada; <=0 desactiva el límite
}

// New construye el servicio de caché. repo es el índice persistente; ytdlp el
// binario (vacío ⇒ "yt-dlp"); dir el directorio raíz de caché; maxBytes el
// límite de tamaño total (<=0 sin límite) y maxAge la antigüedad máxima por
// entrada (<=0 sin límite).
func New(repo *storage.CacheRepo, ytdlp, dir string, maxBytes int64, maxAge time.Duration) *Service {
	if ytdlp == "" {
		ytdlp = "yt-dlp"
	}
	return &Service{
		idx:      newIndex(repo),
		ytdlp:    ytdlp,
		dir:      dir,
		maxBytes: maxBytes,
		maxAge:   maxAge,
	}
}

// audioDir devuelve el subdirectorio donde se guardan los archivos de audio.
func (s *Service) audioDir() string { return filepath.Join(s.dir, "audio") }

// coversDir devuelve el subdirectorio donde el resolutor de portadas (Fase 4)
// cachea las imágenes de cover y los marcadores negativos .miss. La eliminación
// de esta caché se apoya en el ciclo de vida del resto de la caché (Evict/Clear)
// para mantener un único punto de limpieza.
func (s *Service) coversDir() string { return filepath.Join(s.dir, "covers") }

// Lookup devuelve la ruta del archivo cacheado de la pista id si existe una
// entrada válida. Valida que el archivo exista y no esté vacío; si falta o está
// corrupto (tamaño cero), invalida la entrada del índice y devuelve ok=false,
// de modo que la pista se vuelva a descargar/streamear. Un acierto renueva la
// marca de uso (last_used) para la política de expiración.
func (s *Service) Lookup(id string) (string, bool) {
	entry, found, err := s.idx.get(id)
	if err != nil || !found {
		return "", false
	}

	info, err := os.Stat(entry.Path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		// Archivo ausente o corrupto: invalidar la entrada y forzar re-descarga.
		_ = s.idx.remove(id)
		return "", false
	}

	_ = s.idx.touch(id)
	return entry.Path, true
}

// Download extrae el audio de la pista r mediante `yt-dlp -x --write-thumbnail`
// al directorio de caché, registra la entrada en el índice y ejecuta la
// expiración post-descarga. Devuelve la ruta del archivo de audio resultante.
func (s *Service) Download(ctx context.Context, r search.Result) (string, error) {
	if r.ID == "" {
		return "", fmt.Errorf("download: empty video id")
	}
	if err := os.MkdirAll(s.audioDir(), 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	// yt-dlp resuelve la extensión final tras la extracción; se usa la plantilla
	// %(ext)s y luego se localiza el archivo producido para conocerla.
	outTmpl := filepath.Join(s.audioDir(), r.ID+".%(ext)s")
	args := []string{
		"-x",                // extraer audio
		"--write-thumbnail", // guardar la miniatura junto al audio (artwork)
		"--no-warnings",
		"-o", outTmpl,
		r.URL(),
	}
	cmd := exec.CommandContext(ctx, s.ytdlp, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	path, ext, err := s.findAudioFile(r.ID)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat cached audio %q: %w", path, err)
	}

	if err := s.idx.record(r, path, info.Size(), ext); err != nil {
		return "", err
	}

	// Expiración post-descarga: recupera espacio en cuanto se supera el límite.
	if err := s.Evict(); err != nil {
		return "", err
	}
	return path, nil
}

// ThumbPath devuelve la ruta de la miniatura cacheada (escrita por
// `--write-thumbnail` junto al audio) para la pista id, si existe un archivo no
// vacío. Permite reutilizar la portada local en vez de re-descargar la miniatura
// remota; ante una ausencia devuelve ok=false para que la capa de artwork caiga
// a la URL de YouTube.
func (s *Service) ThumbPath(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	matches, err := filepath.Glob(filepath.Join(s.audioDir(), id+".*"))
	if err != nil {
		return "", false
	}
	for _, m := range matches {
		ext := strings.TrimPrefix(filepath.Ext(m), ".")
		if !isThumbnailExt(ext) {
			continue
		}
		if info, statErr := os.Stat(m); statErr == nil && !info.IsDir() && info.Size() > 0 {
			return m, true
		}
	}
	return "", false
}

// findAudioFile localiza el archivo de audio producido por yt-dlp para id,
// descartando la miniatura. Devuelve la ruta y la extensión (sin punto).
func (s *Service) findAudioFile(id string) (path, ext string, err error) {
	matches, globErr := filepath.Glob(filepath.Join(s.audioDir(), id+".*"))
	if globErr != nil {
		return "", "", fmt.Errorf("glob cached audio: %w", globErr)
	}
	for _, m := range matches {
		e := strings.TrimPrefix(filepath.Ext(m), ".")
		if isThumbnailExt(e) {
			continue
		}
		return m, e, nil
	}
	return "", "", fmt.Errorf("download: yt-dlp produced no audio file for %q", id)
}

// isThumbnailExt indica si ext corresponde a una miniatura escrita por
// --write-thumbnail y no al archivo de audio.
func isThumbnailExt(ext string) bool {
	switch strings.ToLower(ext) {
	case "jpg", "jpeg", "png", "webp", "gif":
		return true
	default:
		return false
	}
}

// Evict aplica la política de expiración: elimina primero las entradas que
// superan la antigüedad máxima (maxAge) y luego, si el tamaño total excede
// maxBytes, borra las entradas menos usadas hasta quedar bajo el límite. Borra
// tanto el archivo como la fila del índice. Límites <=0 desactivan esa
// dimensión. Una entrada cuyo archivo ya no existe se elimina del índice.
func (s *Service) Evict() error {
	entries, err := s.idx.oldest()
	if err != nil {
		return err
	}

	now := time.Now()
	var total int64
	dropped := false
	live := make([]storage.CacheEntry, 0, len(entries))

	// Paso 1: expiración por antigüedad (y limpieza de archivos faltantes).
	for _, e := range entries {
		info, statErr := os.Stat(e.Path)
		if statErr != nil {
			// Archivo ausente: la entrada de índice es basura, eliminarla.
			if err := s.dropEntry(e.VideoID, e.Path); err != nil {
				return err
			}
			dropped = true
			continue
		}
		if s.maxAge > 0 {
			if created, perr := parseDBTime(e.CreatedAt); perr == nil && now.Sub(created) > s.maxAge {
				if err := s.dropEntry(e.VideoID, e.Path); err != nil {
					return err
				}
				dropped = true
				continue
			}
		}
		total += info.Size()
		live = append(live, e)
	}

	// Paso 2: expiración por tamaño. live ya viene ordenada por last_used ASC
	// (menos usadas primero), que es el orden de borrado deseado.
	if s.maxBytes > 0 {
		for _, e := range live {
			if total <= s.maxBytes {
				break
			}
			if err := s.dropEntry(e.VideoID, e.Path); err != nil {
				return err
			}
			dropped = true
			total -= e.SizeBytes
		}
	}

	// La caché de portadas (covers/) está direccionada por contenido y no
	// indexada por entrada, así que su limpieza se apoya en el ciclo de vida de
	// la caché de audio: cuando la expiración descarta entradas, se purgan
	// también las portadas para no dejar imágenes huérfanas. Se regeneran bajo
	// demanda en la siguiente reproducción.
	if dropped {
		if err := os.RemoveAll(s.coversDir()); err != nil {
			return fmt.Errorf("evict covers dir: %w", err)
		}
	}
	return nil
}

// Sweep ejecuta la expiración una vez al arrancar la aplicación, de modo que se
// recupere espacio entre sesiones y tras bajar los límites de tamaño/antigüedad.
func (s *Service) Sweep() error { return s.Evict() }

// Clear vacía la caché: borra el directorio de audio y todas las entradas del
// índice. La aplicación sigue tratando las pistas como no cacheadas.
func (s *Service) Clear() error {
	entries, err := s.idx.oldest()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := s.idx.remove(e.VideoID); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(s.audioDir()); err != nil {
		return fmt.Errorf("clear cache dir: %w", err)
	}
	// La caché de portadas (covers/) se vacía junto con el audio: comparte el
	// ciclo de vida de la caché y se regenera bajo demanda.
	if err := os.RemoveAll(s.coversDir()); err != nil {
		return fmt.Errorf("clear covers dir: %w", err)
	}
	return nil
}

// dropEntry elimina el archivo de audio (ignorando si ya no existe), la
// miniatura cacheada junto a él (si la hay) y su fila de índice.
func (s *Service) dropEntry(videoID, path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cached file %q: %w", path, err)
	}
	// La miniatura escrita por --write-thumbnail vive junto al audio y no está
	// indexada; se borra aquí para que la expiración no la deje huérfana.
	if thumb, ok := s.ThumbPath(videoID); ok {
		if err := os.Remove(thumb); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove cached thumbnail %q: %w", thumb, err)
		}
	}
	return s.idx.remove(videoID)
}

// parseDBTime interpreta una marca de tiempo escrita por datetime('now') de
// SQLite (formato "2006-01-02 15:04:05" en UTC).
func parseDBTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", strings.TrimSpace(s))
}
