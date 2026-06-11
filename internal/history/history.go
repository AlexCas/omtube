// Package history registra las pistas reproducidas. Se respalda en la capa de
// almacenamiento SQLite (internal/storage) y conserva su forma pública
// (Entry/Add/Entries) para minimizar los cambios en main.go y la UI. En el
// primer arranque importa, una sola vez, el archivo legado history.json.
package history

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sort"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// Entry es una pista reproducida con su marca de tiempo. Conserva el shape JSON
// legado para poder importar history.json.
type Entry struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Uploader string    `json:"uploader"`
	PlayedAt time.Time `json:"played_at"`
}

// History persiste el historial de reproducción sobre el repositorio SQLite.
type History struct {
	repo   *storage.HistoryRepo
	tracks *storage.TrackRepo
	now    func() time.Time
}

// Load construye el historial respaldado por los repositorios de almacenamiento.
// Si legacyJSONPath no está vacío y existe, y la tabla de historial está vacía,
// importa las entradas una sola vez y renombra el archivo a "<path>.bak" (nunca
// lo borra). Un archivo legado ausente no es error.
func Load(repo *storage.HistoryRepo, tracks *storage.TrackRepo, legacyJSONPath string) (*History, error) {
	h := &History{repo: repo, tracks: tracks, now: time.Now}
	if legacyJSONPath != "" {
		if err := h.importLegacyJSON(legacyJSONPath); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// Entries devuelve las entradas registradas (más antiguas primero) para
// conservar el contrato de Fase 1.
func (h *History) Entries() []Entry {
	entries, err := h.entries(true)
	if err != nil {
		return nil
	}
	return entries
}

// Browse devuelve las entradas de historial más recientes primero, para la vista
// navegable de la biblioteca.
func (h *History) Browse() []Entry {
	entries, err := h.entries(false)
	if err != nil {
		return nil
	}
	return entries
}

// Add registra una pista como reproducida ahora. Garantiza que la pista exista
// en el repositorio de pistas (FK) antes de insertar el historial.
func (h *History) Add(r search.Result) error {
	if err := h.tracks.Upsert(r); err != nil {
		return err
	}
	return h.repo.Insert(r.ID, h.now())
}

// entries lee el historial del repositorio. oldestFirst=true invierte el orden
// recent-first del repositorio para conservar el contrato de Entries().
func (h *History) entries(oldestFirst bool) ([]Entry, error) {
	rows, err := h.repo.List(0)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for _, e := range rows {
		out = append(out, Entry{
			ID:       e.Track.ID,
			Title:    e.Track.Title,
			Uploader: e.Track.Uploader,
			PlayedAt: e.PlayedAt,
		})
	}
	if oldestFirst {
		// repo.List devuelve recent-first; invertimos a oldest-first.
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}

// importLegacyJSON realiza la importación única del archivo history.json a la
// base de datos. Es idempotente: si la tabla ya tiene entradas, no hace nada.
func (h *History) importLegacyJSON(path string) error {
	existing, err := h.repo.List(1)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil // ya hay historial en la BD: no reimportar
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // sin archivo legado: nada que importar
		}
		return err
	}
	if len(data) == 0 {
		return h.backupLegacy(path)
	}

	var legacy []Entry
	if err := json.Unmarshal(data, &legacy); err != nil {
		// Un history.json malformado NO debe abortar el arranque: la app
		// quedaría inservible de forma permanente porque el archivo solo se
		// respalda tras una importación exitosa. Lo respaldamos a ".bak" para no
		// reprocesarlo y continuamos con historial vacío.
		return h.backupLegacy(path)
	}

	// Importar en orden cronológico (más antiguas primero) para preservar el
	// orden original al consultarlas de nuevo.
	sort.SliceStable(legacy, func(i, j int) bool {
		return legacy[i].PlayedAt.Before(legacy[j].PlayedAt)
	})

	// La importación masiva es atómica (todo-o-nada): si falla a mitad de
	// camino, el rollback deja la tabla vacía y NO respaldamos el archivo, de
	// modo que un fallo transitorio pueda reintentarse en el próximo arranque.
	tx, err := h.repo.DB().Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op si ya se hizo Commit

	for _, e := range legacy {
		if err := h.tracks.UpsertTx(tx, search.Result{
			ID:       e.ID,
			Title:    e.Title,
			Uploader: e.Uploader,
		}); err != nil {
			return err
		}
		if err := h.repo.InsertTx(tx, e.ID, e.PlayedAt); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Solo tras un commit exitoso renombramos el archivo a ".bak".
	return h.backupLegacy(path)
}

// backupLegacy renombra el archivo legado a "<path>.bak" para conservarlo y
// evitar reimportaciones.
func (h *History) backupLegacy(path string) error {
	if err := os.Rename(path, path+".bak"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
