// Package playlist implementa la lógica de dominio de playlists sobre los
// repositorios de almacenamiento: validación de nombres, reglas de duplicados y
// reproducción como cola.
package playlist

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// Errores de dominio de playlists.
var (
	// ErrEmptyName se devuelve cuando el nombre está vacío o en blanco.
	ErrEmptyName = errors.New("playlist: el nombre no puede estar vacío")
	// ErrDuplicateName se devuelve cuando ya existe una playlist con ese nombre.
	ErrDuplicateName = errors.New("playlist: ya existe una playlist con ese nombre")
	// ErrNotFound se devuelve cuando la playlist no existe.
	ErrNotFound = errors.New("playlist: la playlist no existe")
	// ErrEmptyPlaylist se devuelve al reproducir una playlist sin pistas.
	ErrEmptyPlaylist = errors.New("playlist: la playlist está vacía")
)

// Service envuelve los repositorios de playlists y pistas con validación.
type Service struct {
	playlists *storage.PlaylistRepo
	tracks    *storage.TrackRepo
}

// New construye el servicio de playlists.
func New(playlists *storage.PlaylistRepo, tracks *storage.TrackRepo) *Service {
	return &Service{playlists: playlists, tracks: tracks}
}

// Create crea una playlist con nombre no vacío y único. Devuelve ErrEmptyName o
// ErrDuplicateName según corresponda.
func (s *Service) Create(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrEmptyName
	}
	exists, err := s.nameExists(name, 0)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, ErrDuplicateName
	}
	return s.playlists.Create(name)
}

// Rename renombra una playlist existente aplicando las mismas reglas que Create
// (no vacío, único). Devuelve ErrNotFound si el id no existe.
func (s *Service) Rename(id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	found, err := s.exists(id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	collision, err := s.nameExists(name, id)
	if err != nil {
		return err
	}
	if collision {
		return ErrDuplicateName
	}
	if err := s.playlists.Rename(id, name); err != nil {
		return s.mapNotFound(err)
	}
	return nil
}

// Delete borra una playlist y su membresía. Devuelve ErrNotFound si no existe.
func (s *Service) Delete(id int64) error {
	if err := s.playlists.Delete(id); err != nil {
		return s.mapNotFound(err)
	}
	return nil
}

// List devuelve todas las playlists.
func (s *Service) List() ([]storage.Playlist, error) {
	return s.playlists.List()
}

// Add añade una pista a la playlist. Garantiza que la pista exista en el
// repositorio de pistas (upsert) y evita duplicados dentro de la playlist.
// Devuelve ErrNotFound si la playlist no existe.
func (s *Service) Add(id int64, track search.Result) error {
	found, err := s.exists(id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	if err := s.tracks.Upsert(track); err != nil {
		return err
	}
	return s.playlists.Add(id, track.ID)
}

// Remove quita una pista de la playlist. Quitar una pista ausente es un no-op.
// Devuelve ErrNotFound si la playlist no existe.
func (s *Service) Remove(id int64, videoID string) error {
	found, err := s.exists(id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return s.playlists.Remove(id, videoID)
}

// Tracks devuelve las pistas de la playlist en orden. Devuelve ErrNotFound si
// la playlist no existe.
func (s *Service) Tracks(id int64) ([]search.Result, error) {
	found, err := s.exists(id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return s.playlists.Tracks(id)
}

// Play devuelve las pistas de la playlist en orden para cargarlas en la cola.
// Devuelve ErrNotFound si la playlist no existe y ErrEmptyPlaylist si no tiene
// pistas.
func (s *Service) Play(id int64) ([]search.Result, error) {
	tracks, err := s.Tracks(id)
	if err != nil {
		return nil, err
	}
	if len(tracks) == 0 {
		return nil, ErrEmptyPlaylist
	}
	return tracks, nil
}

// exists indica si una playlist con ese id existe.
func (s *Service) exists(id int64) (bool, error) {
	list, err := s.playlists.List()
	if err != nil {
		return false, err
	}
	for _, p := range list {
		if p.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// nameExists indica si existe una playlist con ese nombre, excluyendo el id
// dado (excludeID=0 no excluye nada).
func (s *Service) nameExists(name string, excludeID int64) (bool, error) {
	list, err := s.playlists.List()
	if err != nil {
		return false, err
	}
	for _, p := range list {
		if p.ID != excludeID && p.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// mapNotFound traduce el sql.ErrNoRows del repositorio a ErrNotFound de dominio.
func (s *Service) mapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
