package storage

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestCacheRepoRoundTrip(t *testing.T) {
	db := tmpDB(t)
	if err := db.Tracks().Upsert(search.Result{ID: "x", Title: "X", Uploader: "u", Duration: 10}); err != nil {
		t.Fatalf("Upsert track: %v", err)
	}
	repo := db.Cache()

	if _, found, err := repo.Get("x"); err != nil || found {
		t.Fatalf("Get before upsert = (found=%v, err=%v), want (false, nil)", found, err)
	}

	if err := repo.Upsert(CacheEntry{VideoID: "x", Path: "/c/x.opus", SizeBytes: 4096, Ext: "opus"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, found, err := repo.Get("x")
	if err != nil || !found {
		t.Fatalf("Get = (found=%v, err=%v), want (true, nil)", found, err)
	}
	if got.Path != "/c/x.opus" || got.SizeBytes != 4096 || got.Ext != "opus" {
		t.Fatalf("Get = %+v, want path/size/ext set", got)
	}

	total, err := repo.TotalBytes()
	if err != nil || total != 4096 {
		t.Fatalf("TotalBytes = (%d, %v), want (4096, nil)", total, err)
	}

	// Upsert del mismo id actualiza sin duplicar.
	if err := repo.Upsert(CacheEntry{VideoID: "x", Path: "/c/x.m4a", SizeBytes: 8192, Ext: "m4a"}); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	got, _, _ = repo.Get("x")
	if got.Ext != "m4a" || got.SizeBytes != 8192 {
		t.Fatalf("Get after update = %+v, want m4a/8192", got)
	}

	list, err := repo.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("List = (%d entries, %v), want (1, nil)", len(list), err)
	}

	if err := repo.Touch("x"); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	// Touch de inexistente: no-op sin error.
	if err := repo.Touch("nope"); err != nil {
		t.Fatalf("Touch missing: %v", err)
	}

	if err := repo.Delete("x"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Delete de inexistente: no-op sin error.
	if err := repo.Delete("x"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
	if _, found, _ := repo.Get("x"); found {
		t.Fatal("Get after delete: found = true, want false")
	}
}

func TestCacheRepoCascadeOnTrackDelete(t *testing.T) {
	db := tmpDB(t)
	if err := db.Tracks().Upsert(search.Result{ID: "x", Title: "X"}); err != nil {
		t.Fatalf("Upsert track: %v", err)
	}
	if err := db.Cache().Upsert(CacheEntry{VideoID: "x", Path: "/c/x.opus", SizeBytes: 1}); err != nil {
		t.Fatalf("Upsert cache: %v", err)
	}
	if _, err := db.SQL().Exec(`DELETE FROM tracks WHERE video_id = ?`, "x"); err != nil {
		t.Fatalf("delete track: %v", err)
	}
	if _, found, _ := db.Cache().Get("x"); found {
		t.Fatal("cache entry should cascade-delete with its track")
	}
}

func TestCacheRepoTotalBytesEmpty(t *testing.T) {
	total, err := tmpDB(t).Cache().TotalBytes()
	if err != nil || total != 0 {
		t.Fatalf("TotalBytes empty = (%d, %v), want (0, nil)", total, err)
	}
}
