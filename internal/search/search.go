// Package search define el contrato de búsqueda de canciones y sus resultados.
package search

import "context"

// Result representa una canción encontrada en YouTube.
type Result struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	Duration int    `json:"duration"` // segundos
}

// URL devuelve la URL de YouTube de la pista.
func (r Result) URL() string { return "https://www.youtube.com/watch?v=" + r.ID }

// Searcher busca canciones por texto libre.
type Searcher interface {
	// Search devuelve hasta n resultados para la consulta q.
	Search(ctx context.Context, q string, n int) ([]Result, error)
}
