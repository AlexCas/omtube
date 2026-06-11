package storage

import (
	"testing"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestTrackRepoUpsertAndGet(t *testing.T) {
	repo := tmpDB(t).Tracks()

	track := search.Result{ID: "abc123", Title: "Song", Uploader: "Artist", Duration: 200}
	if err := repo.Upsert(track); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, found, err := repo.Get("abc123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get: found = false, want true")
	}
	if got != track {
		t.Fatalf("Get = %+v, want %+v", got, track)
	}

	// Upsert con el mismo id actualiza los metadatos sin duplicar.
	updated := search.Result{ID: "abc123", Title: "Song (Remix)", Uploader: "Artist", Duration: 250}
	if err := repo.Upsert(updated); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	got, _, err = repo.Get("abc123")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got != updated {
		t.Fatalf("Get after update = %+v, want %+v", got, updated)
	}
}

// TestTrackRepoUpsertDoesNotBlankFields verifica que un upsert con campos vacíos
// (p. ej. una pista seleccionada desde el historial, que no almacena duración) NO
// degrada los metadatos ya guardados del registro compartido.
func TestTrackRepoUpsertDoesNotBlankFields(t *testing.T) {
	repo := tmpDB(t).Tracks()

	full := search.Result{ID: "vid1", Title: "Song", Uploader: "Artist", Duration: 300}
	if err := repo.Upsert(full); err != nil {
		t.Fatalf("Upsert full: %v", err)
	}

	// Upsert del mismo video_id con campos vacíos (duración 0, título/uploader "").
	if err := repo.Upsert(search.Result{ID: "vid1"}); err != nil {
		t.Fatalf("Upsert blank: %v", err)
	}

	got, found, err := repo.Get("vid1")
	if err != nil || !found {
		t.Fatalf("Get: found=%v err=%v", found, err)
	}
	if got != full {
		t.Fatalf("upsert vacío degradó el registro: got %+v, want %+v", got, full)
	}

	// Un upsert con duración real (>0) sí debe actualizar.
	if err := repo.Upsert(search.Result{ID: "vid1", Title: "Song", Uploader: "Artist", Duration: 420}); err != nil {
		t.Fatalf("Upsert update duration: %v", err)
	}
	got, _, _ = repo.Get("vid1")
	if got.Duration != 420 {
		t.Fatalf("duración no actualizada: got %d, want 420", got.Duration)
	}
}

func TestTrackRepoGetMissing(t *testing.T) {
	repo := tmpDB(t).Tracks()

	got, found, err := repo.Get("nope")
	if err != nil {
		t.Fatalf("Get missing returned error: %v", err)
	}
	if found {
		t.Fatal("Get missing: found = true, want false")
	}
	if got != (search.Result{}) {
		t.Fatalf("Get missing: result = %+v, want zero value", got)
	}
}
