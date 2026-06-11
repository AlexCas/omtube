package storage

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestFavoriteRepoRoundTrip(t *testing.T) {
	db := tmpDB(t)
	tracks := db.Tracks()
	repo := db.Favorites()

	track := search.Result{ID: "x", Title: "X", Uploader: "u", Duration: 10}
	if err := tracks.Upsert(track); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if ok, err := repo.Exists("x"); err != nil || ok {
		t.Fatalf("Exists before add = (%v, %v), want (false, nil)", ok, err)
	}

	if err := repo.Add("x"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Add idempotente.
	if err := repo.Add("x"); err != nil {
		t.Fatalf("Add idempotent: %v", err)
	}

	if ok, err := repo.Exists("x"); err != nil || !ok {
		t.Fatalf("Exists after add = (%v, %v), want (true, nil)", ok, err)
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != "x" {
		t.Fatalf("List = %+v, want one favorite x", list)
	}

	if err := repo.Remove("x"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// Remove de no-favorito: no-op sin error.
	if err := repo.Remove("x"); err != nil {
		t.Fatalf("Remove non-favorite: %v", err)
	}
	if ok, _ := repo.Exists("x"); ok {
		t.Fatal("Exists after remove = true, want false")
	}
}

func TestFavoriteRepoListEmpty(t *testing.T) {
	list, err := tmpDB(t).Favorites().List()
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("List empty = %+v, want empty", list)
	}
}
