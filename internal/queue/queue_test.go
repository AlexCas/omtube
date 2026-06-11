package queue

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func track(id string) search.Result { return search.Result{ID: id, Title: id} }

func TestEmptyQueue(t *testing.T) {
	q := New()
	if _, ok := q.Current(); ok {
		t.Fatal("cola vacía no debe tener pista actual")
	}
	if q.Next() || q.Prev() {
		t.Fatal("Next/Prev en cola vacía deben devolver false")
	}
}

func TestAddSetsCurrentOnEmpty(t *testing.T) {
	q := New()
	q.Add(track("a"))
	cur, ok := q.Current()
	if !ok || cur.ID != "a" {
		t.Fatalf("la primera pista debe ser la actual, got %+v ok=%v", cur, ok)
	}
	q.Add(track("b"))
	if cur, _ := q.Current(); cur.ID != "a" {
		t.Fatalf("encolar no debe cambiar la actual, got %s", cur.ID)
	}
	if q.Len() != 2 {
		t.Fatalf("Len = %d, want 2", q.Len())
	}
}

func TestNextPrev(t *testing.T) {
	q := New()
	for _, id := range []string{"a", "b", "c"} {
		q.Add(track(id))
	}
	if !q.Next() {
		t.Fatal("Next debe avanzar")
	}
	if cur, _ := q.Current(); cur.ID != "b" {
		t.Fatalf("tras Next, actual = %s, want b", cur.ID)
	}
	if !q.Next() {
		t.Fatal("Next debe avanzar a c")
	}
	if q.Next() {
		t.Fatal("Next en la última debe devolver false (sin wrap)")
	}
	if cur, _ := q.Current(); cur.ID != "c" {
		t.Fatalf("debe quedarse en c, got %s", cur.ID)
	}
	if !q.Prev() {
		t.Fatal("Prev debe retroceder")
	}
	if cur, _ := q.Current(); cur.ID != "b" {
		t.Fatalf("tras Prev, actual = %s, want b", cur.ID)
	}
	q.Prev()
	if q.Prev() {
		t.Fatal("Prev en la primera debe devolver false")
	}
	if cur, _ := q.Current(); cur.ID != "a" {
		t.Fatalf("debe quedarse en a, got %s", cur.ID)
	}
}
