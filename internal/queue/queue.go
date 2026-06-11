// Package queue gestiona la cola de reproducción y la pista actual.
package queue

import "github.com/alexcasdev/terminaltube/internal/search"

// Queue es una lista ordenada de pistas con un cursor a la pista actual.
// El cero-value es una cola vacía lista para usar.
type Queue struct {
	items []search.Result
	idx   int // índice de la pista actual; -1 si no hay actual
}

// New crea una cola vacía.
func New() *Queue { return &Queue{idx: -1} }

// Add encola una pista al final. Si la cola estaba vacía, esa pista pasa a ser la
// actual.
func (q *Queue) Add(r search.Result) {
	q.items = append(q.items, r)
	if q.idx == -1 {
		q.idx = 0
	}
}

// Items devuelve la lista de pistas (no modificar).
func (q *Queue) Items() []search.Result { return q.items }

// Len devuelve el número de pistas en la cola.
func (q *Queue) Len() int { return len(q.items) }

// Index devuelve el índice de la pista actual (-1 si no hay).
func (q *Queue) Index() int { return q.idx }

// Current devuelve la pista actual y true, o false si no hay pista reproducible.
func (q *Queue) Current() (search.Result, bool) {
	if q.idx < 0 || q.idx >= len(q.items) {
		return search.Result{}, false
	}
	return q.items[q.idx], true
}

// Next avanza a la siguiente pista. Devuelve true si avanzó; false si ya estaba en
// la última (sin wrap).
func (q *Queue) Next() bool {
	if q.idx >= 0 && q.idx < len(q.items)-1 {
		q.idx++
		return true
	}
	return false
}

// Prev retrocede a la pista anterior. Devuelve true si retrocedió; false si ya
// estaba en la primera.
func (q *Queue) Prev() bool {
	if q.idx > 0 {
		q.idx--
		return true
	}
	return false
}
