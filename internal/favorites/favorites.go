// Package favorites implementa la lógica de dominio de favoritos sobre los
// repositorios de almacenamiento: alternar el estado de favorito de una pista y
// listar las favoritas.
package favorites

import (
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// Service envuelve los repositorios de favoritos y pistas.
type Service struct {
	favorites *storage.FavoriteRepo
	tracks    *storage.TrackRepo
}

// New construye el servicio de favoritos.
func New(favorites *storage.FavoriteRepo, tracks *storage.TrackRepo) *Service {
	return &Service{favorites: favorites, tracks: tracks}
}

// Toggle alterna el estado de favorito de una pista. Si no era favorita la
// marca y devuelve true; si ya era favorita la desmarca y devuelve false. Las
// operaciones son idempotentes a nivel de almacenamiento. Garantiza que la
// pista exista en el repositorio de pistas antes de marcarla.
func (s *Service) Toggle(track search.Result) (bool, error) {
	fav, err := s.favorites.Exists(track.ID)
	if err != nil {
		return false, err
	}
	if fav {
		if err := s.favorites.Remove(track.ID); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := s.tracks.Upsert(track); err != nil {
		return false, err
	}
	if err := s.favorites.Add(track.ID); err != nil {
		return false, err
	}
	return true, nil
}

// IsFavorite indica si la pista está marcada como favorita.
func (s *Service) IsFavorite(videoID string) (bool, error) {
	return s.favorites.Exists(videoID)
}

// List devuelve las pistas favoritas. Sin favoritos devuelve una lista vacía
// sin error.
func (s *Service) List() ([]search.Result, error) {
	return s.favorites.List()
}
