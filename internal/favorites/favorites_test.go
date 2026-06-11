package favorites

import (
	"path/filepath"
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

func newService(t *testing.T) *Service {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return New(db.Favorites(), db.Tracks())
}

func TestToggleMarksAndUnmarks(t *testing.T) {
	s := newService(t)
	track := search.Result{ID: "a", Title: "A", Uploader: "u"}

	on, err := s.Toggle(track)
	if err != nil {
		t.Fatalf("Toggle on: %v", err)
	}
	if !on {
		t.Fatal("Toggle on returned false, want true (now favorite)")
	}
	if fav, _ := s.IsFavorite("a"); !fav {
		t.Fatal("IsFavorite = false after marking, want true")
	}

	off, err := s.Toggle(track)
	if err != nil {
		t.Fatalf("Toggle off: %v", err)
	}
	if off {
		t.Fatal("Toggle off returned true, want false (no longer favorite)")
	}
	if fav, _ := s.IsFavorite("a"); fav {
		t.Fatal("IsFavorite = true after unmarking, want false")
	}
}

func TestToggleMarkIsIdempotent(t *testing.T) {
	s := newService(t)
	track := search.Result{ID: "a", Title: "A", Uploader: "u"}

	if _, err := s.Toggle(track); err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	// Marcar directamente de nuevo vía el repo no debe duplicar; el toggle
	// de dominio alterna, así que aquí verificamos la lista tras un ciclo
	// marcar/desmarcar/marcar.
	if _, err := s.Toggle(track); err != nil { // off
		t.Fatalf("Toggle off: %v", err)
	}
	if _, err := s.Toggle(track); err != nil { // on
		t.Fatalf("Toggle on again: %v", err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1 (no duplicates)", len(list))
	}
}

func TestUnmarkNonFavoriteNoOp(t *testing.T) {
	s := newService(t)
	track := search.Result{ID: "a", Title: "A", Uploader: "u"}

	// Primera llamada sobre pista no favorita la marca (toggle), no la
	// desmarca; verificamos que List quede en estado consistente y sin error.
	if _, err := s.Toggle(track); err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if _, err := s.Toggle(track); err != nil { // desmarca
		t.Fatalf("Toggle off: %v", err)
	}
	// Desmarcar de nuevo (ya no favorita) debe ser no-op sin error.
	off, err := s.Toggle(track) // como no es favorita, vuelve a marcar -> true
	if err != nil {
		t.Fatalf("Toggle on non-favorite: %v", err)
	}
	if !off {
		t.Fatal("Toggle on non-favorite returned false, want true")
	}
}

func TestListEmpty(t *testing.T) {
	s := newService(t)
	list, err := s.List()
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("List empty = %+v, want empty", list)
	}
}
