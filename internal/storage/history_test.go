package storage

import (
	"testing"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
)

func TestHistoryRepoInsertAndListRecentFirst(t *testing.T) {
	db := tmpDB(t)
	tracks := db.Tracks()
	repo := db.History()

	older := search.Result{ID: "old", Title: "Old", Uploader: "u"}
	newer := search.Result{ID: "new", Title: "New", Uploader: "u"}
	for _, tr := range []search.Result{older, newer} {
		if err := tracks.Upsert(tr); err != nil {
			t.Fatalf("Upsert %s: %v", tr.ID, err)
		}
	}

	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	if err := repo.Insert("old", base); err != nil {
		t.Fatalf("Insert old: %v", err)
	}
	if err := repo.Insert("new", base.Add(time.Hour)); err != nil {
		t.Fatalf("Insert new: %v", err)
	}

	got, err := repo.List(0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List len = %d, want 2", len(got))
	}
	if got[0].Track.ID != "new" || got[1].Track.ID != "old" {
		t.Fatalf("List order = [%s, %s], want [new, old] (recent-first)", got[0].Track.ID, got[1].Track.ID)
	}
	if !got[0].PlayedAt.Equal(base.Add(time.Hour)) {
		t.Fatalf("PlayedAt = %v, want %v", got[0].PlayedAt, base.Add(time.Hour))
	}
}

func TestHistoryRepoListLimit(t *testing.T) {
	db := tmpDB(t)
	if err := db.Tracks().Upsert(search.Result{ID: "t", Title: "T", Uploader: "u"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	repo := db.History()
	base := time.Now().UTC()
	for i := 0; i < 5; i++ {
		if err := repo.Insert("t", base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("Insert %d: %v", i, err)
		}
	}

	got, err := repo.List(2)
	if err != nil {
		t.Fatalf("List(2): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List(2) len = %d, want 2", len(got))
	}
}

// TestTxInsertsAreAllOrNothing verifica que las variantes tx-scoped (UpsertTx /
// InsertTx) son atómicas: si la transacción se revierte, ni la pista ni el
// historial persisten (evidencia de la importación masiva transaccional).
func TestTxInsertsAreAllOrNothing(t *testing.T) {
	db := tmpDB(t)
	tracks := db.Tracks()
	hist := db.History()

	tx, err := hist.DB().Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := tracks.UpsertTx(tx, search.Result{ID: "x", Title: "X", Uploader: "u"}); err != nil {
		t.Fatalf("UpsertTx: %v", err)
	}
	if err := hist.InsertTx(tx, "x", time.Now()); err != nil {
		t.Fatalf("InsertTx: %v", err)
	}
	// Simulamos un fallo a mitad de la importación: rollback.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if _, found, err := tracks.Get("x"); err != nil || found {
		t.Fatalf("tras rollback la pista no debe persistir: found=%v err=%v", found, err)
	}
	got, err := hist.List(0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("tras rollback el historial debe estar vacío, got %d", len(got))
	}

	// Commit posterior sí persiste (camino feliz).
	tx2, err := hist.DB().Begin()
	if err != nil {
		t.Fatalf("Begin 2: %v", err)
	}
	if err := tracks.UpsertTx(tx2, search.Result{ID: "y", Title: "Y", Uploader: "u"}); err != nil {
		t.Fatalf("UpsertTx 2: %v", err)
	}
	if err := hist.InsertTx(tx2, "y", time.Now()); err != nil {
		t.Fatalf("InsertTx 2: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got, err := hist.List(0); err != nil || len(got) != 1 {
		t.Fatalf("tras commit debe haber 1 entrada: len=%d err=%v", len(got), err)
	}
}

func TestHistoryRepoListEmpty(t *testing.T) {
	got, err := tmpDB(t).History().List(0)
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("List empty = %+v, want empty", got)
	}
}
