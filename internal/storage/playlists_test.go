package storage

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestPlaylistRepoCRUDRoundTrip(t *testing.T) {
	db := tmpDB(t)
	repo := db.Playlists()

	id, err := repo.Create("Focus")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Focus" || list[0].ID != id {
		t.Fatalf("List = %+v, want one playlist Focus with id %d", list, id)
	}

	if err := repo.Rename(id, "Deep Work"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	list, _ = repo.List()
	if list[0].Name != "Deep Work" {
		t.Fatalf("after Rename name = %q, want Deep Work", list[0].Name)
	}

	if err := repo.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ = repo.List()
	if len(list) != 0 {
		t.Fatalf("after Delete List = %+v, want empty", list)
	}
}

func TestPlaylistRepoRenameDeleteMissing(t *testing.T) {
	repo := tmpDB(t).Playlists()

	if err := repo.Rename(999, "X"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Rename missing err = %v, want sql.ErrNoRows", err)
	}
	if err := repo.Delete(999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Delete missing err = %v, want sql.ErrNoRows", err)
	}
}

func TestPlaylistRepoTracksOrderedAndDedup(t *testing.T) {
	db := tmpDB(t)
	tracks := db.Tracks()
	repo := db.Playlists()

	a := search.Result{ID: "a", Title: "A", Uploader: "u"}
	b := search.Result{ID: "b", Title: "B", Uploader: "u"}
	c := search.Result{ID: "c", Title: "C", Uploader: "u"}
	for _, tr := range []search.Result{a, b, c} {
		if err := tracks.Upsert(tr); err != nil {
			t.Fatalf("Upsert %s: %v", tr.ID, err)
		}
	}

	id, _ := repo.Create("PL")
	for _, tr := range []search.Result{a, b, c} {
		if err := repo.Add(id, tr.ID); err != nil {
			t.Fatalf("Add %s: %v", tr.ID, err)
		}
	}

	// Add duplicado: no debe crear entrada extra ni reordenar.
	if err := repo.Add(id, a.ID); err != nil {
		t.Fatalf("Add dup: %v", err)
	}

	got, err := repo.Tracks(id)
	if err != nil {
		t.Fatalf("Tracks: %v", err)
	}
	wantOrder := []string{"a", "b", "c"}
	if len(got) != len(wantOrder) {
		t.Fatalf("Tracks len = %d, want %d (%+v)", len(got), len(wantOrder), got)
	}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Fatalf("Tracks[%d].ID = %q, want %q", i, got[i].ID, id)
		}
	}

	// Remove conserva el orden del resto.
	if err := repo.Remove(id, "b"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	got, _ = repo.Tracks(id)
	want := []string{"a", "c"}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("after Remove Tracks[%d].ID = %q, want %q", i, got[i].ID, id)
		}
	}

	// Remove de pista ausente: no-op sin error.
	if err := repo.Remove(id, "zzz"); err != nil {
		t.Fatalf("Remove missing track: %v", err)
	}
}
