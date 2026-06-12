// Package cache descarga el audio de las pistas mediante yt-dlp al directorio
// XDG cache y mantiene un índice en la base de datos compartida (library.db),
// de modo que las repeticiones se sirvan desde el archivo local sin re-resolver
// ni re-descargar. La caché expira por tamaño y antigüedad.
package cache

import (
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// Index es una capa fina sobre storage.CacheRepo. Aísla el servicio de caché
// del SQL y nombra las operaciones en términos del dominio de caché.
type Index struct {
	repo *storage.CacheRepo
}

// newIndex construye el índice sobre el repositorio de caché dado.
func newIndex(repo *storage.CacheRepo) *Index {
	return &Index{repo: repo}
}

// record registra (o actualiza) la entrada de índice de una pista cacheada.
// Inserta la pista padre en tracks dentro de la misma transacción que la entrada
// de caché, de modo que la FK cache_entries→tracks se satisfaga siempre: una
// primera reproducción no depende de que el historial haya insertado la pista
// antes y nunca deja el audio descargado sin indexar.
func (i *Index) record(track search.Result, path string, sizeBytes int64, ext string) error {
	return i.repo.UpsertWithTrack(track, storage.CacheEntry{
		VideoID:   track.ID,
		Path:      path,
		SizeBytes: sizeBytes,
		Ext:       ext,
	})
}

// touch renueva la marca de uso de una pista cacheada.
func (i *Index) touch(videoID string) error { return i.repo.Touch(videoID) }

// get devuelve la entrada de índice de una pista.
func (i *Index) get(videoID string) (storage.CacheEntry, bool, error) {
	return i.repo.Get(videoID)
}

// remove borra la entrada de índice de una pista.
func (i *Index) remove(videoID string) error { return i.repo.Delete(videoID) }

// oldest devuelve todas las entradas ordenadas de la menos usada a la más
// reciente, de modo que la expiración elimine las más antiguas primero.
func (i *Index) oldest() ([]storage.CacheEntry, error) { return i.repo.List() }

// total devuelve el tamaño total en bytes indexado por la caché.
func (i *Index) total() (int64, error) { return i.repo.TotalBytes() }
