package storage

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestLyricsRepoRoundTrip(t *testing.T) {
	db := tmpDB(t)
	if err := db.Tracks().Upsert(search.Result{ID: "x", Title: "X"}); err != nil {
		t.Fatalf("Upsert track: %v", err)
	}
	repo := db.Lyrics()

	if _, found, err := repo.Get("x"); err != nil || found {
		t.Fatalf("Get before upsert = (found=%v, err=%v), want (false, nil)", found, err)
	}

	if err := repo.Upsert(LyricsEntry{VideoID: "x", Synced: true, Body: "[00:01.00]hi"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, found, err := repo.Get("x")
	if err != nil || !found {
		t.Fatalf("Get = (found=%v, err=%v), want (true, nil)", found, err)
	}
	if !got.Synced || got.Body != "[00:01.00]hi" {
		t.Fatalf("Get = %+v, want synced body", got)
	}

	// Upsert del mismo id actualiza (a no sincronizado / texto plano).
	if err := repo.Upsert(LyricsEntry{VideoID: "x", Synced: false, Body: "plain"}); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	got, _, _ = repo.Get("x")
	if got.Synced || got.Body != "plain" {
		t.Fatalf("Get after update = %+v, want plain unsynced", got)
	}
}

func TestLyricsRepoCascadeOnTrackDelete(t *testing.T) {
	db := tmpDB(t)
	if err := db.Tracks().Upsert(search.Result{ID: "x", Title: "X"}); err != nil {
		t.Fatalf("Upsert track: %v", err)
	}
	if err := db.Lyrics().Upsert(LyricsEntry{VideoID: "x", Body: "y"}); err != nil {
		t.Fatalf("Upsert lyrics: %v", err)
	}
	if _, err := db.SQL().Exec(`DELETE FROM tracks WHERE video_id = ?`, "x"); err != nil {
		t.Fatalf("delete track: %v", err)
	}
	if _, found, _ := db.Lyrics().Get("x"); found {
		t.Fatal("lyrics entry should cascade-delete with its track")
	}
}
