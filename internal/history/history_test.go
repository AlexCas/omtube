package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// newHistory abre una base de datos temporal y construye un History respaldado
// por ella, sin importación legada (legacyJSONPath vacío).
func newHistory(t *testing.T) *History {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	h, err := Load(db.History(), db.Tracks(), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return h
}

func TestMissingDataIsEmpty(t *testing.T) {
	h := newHistory(t)
	if len(h.Entries()) != 0 {
		t.Fatalf("historial debe estar vacío, got %d", len(h.Entries()))
	}
	if len(h.Browse()) != 0 {
		t.Fatalf("browse debe estar vacío, got %d", len(h.Browse()))
	}
}

func TestAddPersistsAndOrders(t *testing.T) {
	h := newHistory(t)

	if err := h.Add(search.Result{ID: "a", Title: "Numb", Uploader: "LP"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := h.Add(search.Result{ID: "b", Title: "Faint"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Entries() conserva el orden de Fase 1: más antiguas primero.
	entries := h.Entries()
	if len(entries) != 2 {
		t.Fatalf("se esperaban 2 entradas, got %d", len(entries))
	}
	if entries[0].ID != "a" || entries[1].ID != "b" {
		t.Fatalf("orden oldest-first incorrecto: %+v", entries)
	}
	if entries[0].PlayedAt.IsZero() {
		t.Fatal("PlayedAt no debe ser cero")
	}

	// Browse() devuelve más recientes primero.
	browse := h.Browse()
	if len(browse) != 2 {
		t.Fatalf("se esperaban 2 entradas en browse, got %d", len(browse))
	}
	if browse[0].ID != "b" || browse[1].ID != "a" {
		t.Fatalf("orden recent-first incorrecto: %+v", browse)
	}
}

func TestImportLegacyJSONAndBackup(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "history.json")

	legacy := []Entry{
		{ID: "a", Title: "Numb", Uploader: "LP", PlayedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "b", Title: "Faint", Uploader: "LP", PlayedAt: time.Now().Add(-1 * time.Hour)},
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := storage.Open(filepath.Join(dir, "library.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	h, err := Load(db.History(), db.Tracks(), jsonPath)
	if err != nil {
		t.Fatalf("Load con import: %v", err)
	}

	entries := h.Entries()
	if len(entries) != 2 {
		t.Fatalf("se esperaban 2 entradas importadas, got %d", len(entries))
	}
	if entries[0].ID != "a" || entries[1].ID != "b" {
		t.Fatalf("orden de importación incorrecto: %+v", entries)
	}

	// El archivo original debe conservarse como .bak (nunca borrado).
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("history.json debe renombrarse, aún existe: %v", err)
	}
	if _, err := os.Stat(jsonPath + ".bak"); err != nil {
		t.Fatalf("history.json.bak debe existir: %v", err)
	}
}

// TestCorruptLegacyJSONDoesNotBrickStartup verifica que un history.json
// malformado no aborta el arranque: Load no devuelve error, el historial queda
// vacío y el archivo corrupto se respalda a ".bak" para no reprocesarlo.
func TestCorruptLegacyJSONDoesNotBrickStartup(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "history.json")

	if err := os.WriteFile(jsonPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := storage.Open(filepath.Join(dir, "library.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	// El arranque DEBE tener éxito pese al JSON corrupto.
	h, err := Load(db.History(), db.Tracks(), jsonPath)
	if err != nil {
		t.Fatalf("Load con JSON corrupto no debe fallar: %v", err)
	}
	if got := len(h.Entries()); got != 0 {
		t.Fatalf("historial debe quedar vacío, got %d", got)
	}

	// El archivo corrupto debe haberse respaldado a ".bak" (no reprocesar).
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("history.json corrupto debe renombrarse, aún existe: %v", err)
	}
	if _, err := os.Stat(jsonPath + ".bak"); err != nil {
		t.Fatalf("history.json.bak debe existir: %v", err)
	}
}

func TestImportIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "history.json")
	dbPath := filepath.Join(dir, "library.db")

	legacy := []Entry{{ID: "a", Title: "Numb", PlayedAt: time.Now()}}
	data, _ := json.Marshal(legacy)
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Primera apertura: importa y crea .bak.
	db1, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(db1.History(), db1.Tracks(), jsonPath); err != nil {
		t.Fatalf("primera Load: %v", err)
	}
	_ = db1.Close()

	// Re-crear history.json para detectar una reimportación indebida.
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Segunda apertura: la tabla ya tiene historial → no reimporta.
	db2, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	h, err := Load(db2.History(), db2.Tracks(), jsonPath)
	if err != nil {
		t.Fatalf("segunda Load: %v", err)
	}

	if got := len(h.Entries()); got != 1 {
		t.Fatalf("la importación no es idempotente: se esperaba 1 entrada, got %d", got)
	}
	// El history.json recreado debe quedar intacto (no se renombró de nuevo).
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("history.json recreado no debe tocarse: %v", err)
	}
}
