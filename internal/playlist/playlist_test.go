package playlist

import (
	"errors"
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
	return New(db.Playlists(), db.Tracks())
}

func TestCreateRejectsEmptyName(t *testing.T) {
	s := newService(t)
	for _, name := range []string{"", "   ", "\t"} {
		if _, err := s.Create(name); !errors.Is(err, ErrEmptyName) {
			t.Fatalf("Create(%q) err = %v, want ErrEmptyName", name, err)
		}
	}
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	s := newService(t)
	if _, err := s.Create("Focus"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Create("Focus"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("Create duplicate err = %v, want ErrDuplicateName", err)
	}
}

func TestRenameCollision(t *testing.T) {
	s := newService(t)
	focus, _ := s.Create("Focus")
	if _, err := s.Create("Chill"); err != nil {
		t.Fatalf("Create Chill: %v", err)
	}
	if err := s.Rename(focus, "Chill"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("Rename collision err = %v, want ErrDuplicateName", err)
	}
	// Renombrar al mismo nombre actual no es colisión consigo misma.
	if err := s.Rename(focus, "Focus"); err != nil {
		t.Fatalf("Rename to same name: %v", err)
	}
}

func TestRenameMissing(t *testing.T) {
	s := newService(t)
	if err := s.Rename(999, "New"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Rename missing err = %v, want ErrNotFound", err)
	}
}

func TestAddDuplicateTrackNoOp(t *testing.T) {
	s := newService(t)
	id, _ := s.Create("PL")
	track := search.Result{ID: "a", Title: "A", Uploader: "u"}

	if err := s.Add(id, track); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(id, track); err != nil {
		t.Fatalf("Add dup: %v", err)
	}

	tracks, err := s.Tracks(id)
	if err != nil {
		t.Fatalf("Tracks: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("Tracks len = %d, want 1 (dup add must be no-op)", len(tracks))
	}
}

func TestAddToMissingPlaylist(t *testing.T) {
	s := newService(t)
	if err := s.Add(999, search.Result{ID: "a"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Add missing playlist err = %v, want ErrNotFound", err)
	}
}

func TestPlayEmptyPlaylist(t *testing.T) {
	s := newService(t)
	id, _ := s.Create("Empty")
	if _, err := s.Play(id); !errors.Is(err, ErrEmptyPlaylist) {
		t.Fatalf("Play empty err = %v, want ErrEmptyPlaylist", err)
	}
}

func TestPlayPopulatedPlaylist(t *testing.T) {
	s := newService(t)
	id, _ := s.Create("Focus")
	a := search.Result{ID: "a", Title: "A", Uploader: "u"}
	b := search.Result{ID: "b", Title: "B", Uploader: "u"}
	if err := s.Add(id, a); err != nil {
		t.Fatalf("Add a: %v", err)
	}
	if err := s.Add(id, b); err != nil {
		t.Fatalf("Add b: %v", err)
	}

	tracks, err := s.Play(id)
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
	if len(tracks) != 2 || tracks[0].ID != "a" || tracks[1].ID != "b" {
		t.Fatalf("Play tracks = %+v, want [a, b] in order", tracks)
	}
}
