package cache

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// newRepo abre una library.db temporal, registra las pistas dadas y devuelve el
// repositorio de caché. Sembrar las pistas ya no es obligatorio para evitar la
// FK (Download las inserta dentro de su transacción), pero se conserva para
// cubrir el camino feliz "la pista ya existía".
func newRepo(t *testing.T, tracks ...search.Result) *storage.CacheRepo {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, tr := range tracks {
		if err := db.Tracks().Upsert(tr); err != nil {
			t.Fatalf("Upsert track %q: %v", tr.ID, err)
		}
	}
	return db.Cache()
}

// newDB abre una library.db temporal vacía (sin pistas sembradas) y la devuelve,
// para verificar que Download satisface la FK por sí solo.
func newDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// fakeYtDlp escribe un script que imita a yt-dlp: extrae el id desde la
// plantilla -o y crea un archivo de audio (.opus) con el contenido dado más una
// miniatura (.jpg), para verificar que Download ignora la miniatura.
func fakeYtDlp(t *testing.T, audioContent string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake yt-dlp script is POSIX shell only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "yt-dlp")
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
	case "$1" in
		-o) out="$2"; shift 2 ;;
		*) shift ;;
	esac
done
base="${out%.*}"
printf '%s' '` + audioContent + `' > "${base}.opus"
printf 'thumb' > "${base}.jpg"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}
	return path
}

func TestCacheIndexCRUDRoundTrip(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "vid1", Title: "Song", Uploader: "Artist", Duration: 100})
	idx := newIndex(repo)

	if err := idx.record(search.Result{ID: "vid1"}, "/cache/audio/vid1.opus", 2048, "opus"); err != nil {
		t.Fatalf("record: %v", err)
	}

	e, found, err := idx.get("vid1")
	if err != nil || !found {
		t.Fatalf("get = (found=%v, err=%v), want (true, nil)", found, err)
	}
	if e.Path != "/cache/audio/vid1.opus" || e.SizeBytes != 2048 || e.Ext != "opus" {
		t.Fatalf("get = %+v, want path/size/ext set", e)
	}

	total, err := idx.total()
	if err != nil || total != 2048 {
		t.Fatalf("total = (%d, %v), want (2048, nil)", total, err)
	}

	if err := idx.touch("vid1"); err != nil {
		t.Fatalf("touch: %v", err)
	}

	all, err := idx.oldest()
	if err != nil || len(all) != 1 {
		t.Fatalf("oldest = (%d entries, %v), want (1, nil)", len(all), err)
	}

	if err := idx.remove("vid1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, found, _ := idx.get("vid1"); found {
		t.Fatal("get after remove: found = true, want false")
	}
}

func TestDownloadRecordsIndexAndIgnoresThumbnail(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "vid1", Title: "Song"})
	dir := t.TempDir()
	svc := New(repo, fakeYtDlp(t, "AUDIOBYTES"), dir, 0, 0)

	path, err := svc.Download(context.Background(), search.Result{ID: "vid1"})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if filepath.Ext(path) != ".opus" {
		t.Fatalf("Download path = %q, want .opus audio file (not thumbnail)", path)
	}

	got, ok := svc.Lookup("vid1")
	if !ok || got != path {
		t.Fatalf("Lookup = (%q, %v), want (%q, true)", got, ok, path)
	}
}

func TestDownloadInsertsTrackAndSatisfiesFK(t *testing.T) {
	// C1: con foreign_keys=ON y SIN pista sembrada, Download debe insertar la
	// pista padre y la entrada de caché en la misma transacción, de modo que no
	// haya error de FK y el audio quede indexado y retenido en disco.
	db := newDB(t)
	dir := t.TempDir()
	svc := New(db.Cache(), fakeYtDlp(t, "AUDIOBYTES"), dir, 0, 0)

	r := search.Result{ID: "fresh", Title: "Song", Uploader: "Artist", Duration: 42}
	path, err := svc.Download(context.Background(), r)
	if err != nil {
		t.Fatalf("Download sin pista sembrada = %v, want nil (no FK error)", err)
	}

	// La pista padre quedó registrada con sus metadatos.
	got, found, err := db.Tracks().Get("fresh")
	if err != nil || !found {
		t.Fatalf("Tracks().Get = (found=%v, err=%v), want (true, nil)", found, err)
	}
	if got.Title != "Song" || got.Uploader != "Artist" || got.Duration != 42 {
		t.Fatalf("track guardada = %+v, want metadatos de r", got)
	}

	// La entrada de caché quedó indexada y el archivo retenido.
	if cached, ok := svc.Lookup("fresh"); !ok || cached != path {
		t.Fatalf("Lookup = (%q, %v), want (%q, true): audio quedó sin indexar", cached, ok, path)
	}
	if info, statErr := os.Stat(path); statErr != nil || info.Size() == 0 {
		t.Fatalf("archivo de audio no retenido: stat=%v", statErr)
	}
}

func TestThumbPathReusedAndEvicted(t *testing.T) {
	// W4: Download escribe la miniatura junto al audio; ThumbPath la localiza para
	// reutilizarla. Tras la expiración, la miniatura no debe quedar huérfana.
	db := newDB(t)
	dir := t.TempDir()
	svc := New(db.Cache(), fakeYtDlp(t, "AUDIOBYTES"), dir, 0, 0)

	r := search.Result{ID: "vid1", Title: "Song"}
	if _, err := svc.Download(context.Background(), r); err != nil {
		t.Fatalf("Download: %v", err)
	}

	thumb, ok := svc.ThumbPath("vid1")
	if !ok {
		t.Fatal("ThumbPath = miss, want hit (la miniatura cacheada debe reutilizarse)")
	}
	if filepath.Ext(thumb) != ".jpg" {
		t.Fatalf("ThumbPath = %q, want archivo .jpg", thumb)
	}

	// Expiración: borra la entrada y también su miniatura no indexada.
	if err := svc.dropEntry("vid1", filepath.Join(dir, "audio", "vid1.opus")); err != nil {
		t.Fatalf("dropEntry: %v", err)
	}
	if _, err := os.Stat(thumb); !os.IsNotExist(err) {
		t.Fatal("la miniatura debería borrarse junto con la entrada de caché")
	}
	if _, ok := svc.ThumbPath("vid1"); ok {
		t.Fatal("ThumbPath tras la expiración = hit, want miss")
	}
}

func TestLookupInvalidatesMissingFile(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "vid1", Title: "Song"})
	idx := newIndex(repo)
	svc := New(repo, "", t.TempDir(), 0, 0)

	// Entrada de índice que apunta a un archivo inexistente.
	if err := idx.record(search.Result{ID: "vid1"}, filepath.Join(t.TempDir(), "gone.opus"), 1024, "opus"); err != nil {
		t.Fatalf("record: %v", err)
	}

	if _, ok := svc.Lookup("vid1"); ok {
		t.Fatal("Lookup of missing file = ok, want miss")
	}
	if _, found, _ := idx.get("vid1"); found {
		t.Fatal("missing-file entry was not invalidated from the index")
	}
}

func TestLookupInvalidatesCorruptFile(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "vid1", Title: "Song"})
	idx := newIndex(repo)
	dir := t.TempDir()
	svc := New(repo, "", dir, 0, 0)

	// Archivo de tamaño cero ⇒ corrupto.
	empty := filepath.Join(dir, "vid1.opus")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	if err := idx.record(search.Result{ID: "vid1"}, empty, 0, "opus"); err != nil {
		t.Fatalf("record: %v", err)
	}

	if _, ok := svc.Lookup("vid1"); ok {
		t.Fatal("Lookup of corrupt (empty) file = ok, want miss")
	}
	if _, found, _ := idx.get("vid1"); found {
		t.Fatal("corrupt-file entry was not invalidated from the index")
	}
}

func TestEvictRespectsSizeBudget(t *testing.T) {
	repo := newRepo(t,
		search.Result{ID: "a"},
		search.Result{ID: "b"},
		search.Result{ID: "c"},
	)
	idx := newIndex(repo)
	dir := t.TempDir()
	audioDir := filepath.Join(dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Tres archivos de 100 bytes; tocados en orden a < b < c, así last_used
	// crece a→c y "a" es la menos usada (primera en expirar).
	for _, id := range []string{"a", "b", "c"} {
		p := filepath.Join(audioDir, id+".opus")
		if err := os.WriteFile(p, make([]byte, 100), 0o644); err != nil {
			t.Fatalf("write %s: %v", id, err)
		}
		if err := idx.record(search.Result{ID: id}, p, 100, "opus"); err != nil {
			t.Fatalf("record %s: %v", id, err)
		}
		if err := idx.touch(id); err != nil {
			t.Fatalf("touch %s: %v", id, err)
		}
		time.Sleep(1100 * time.Millisecond) // datetime('now') tiene resolución de 1s
	}

	// Límite 250 bytes: deben quedar como mucho 2 de los 3 (200 bytes), borrando
	// la menos usada ("a").
	svc := New(repo, "", dir, 250, 0)
	if err := svc.Evict(); err != nil {
		t.Fatalf("Evict: %v", err)
	}

	total, err := idx.total()
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total > 250 {
		t.Fatalf("total after evict = %d, want <= 250", total)
	}
	if _, found, _ := idx.get("a"); found {
		t.Fatal("least-used entry 'a' should have been evicted first")
	}
	if _, err := os.Stat(filepath.Join(audioDir, "a.opus")); !os.IsNotExist(err) {
		t.Fatal("evicted file a.opus should have been deleted from disk")
	}
}

func TestSweepEvictsByAge(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "old"})
	idx := newIndex(repo)
	dir := t.TempDir()
	audioDir := filepath.Join(dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := filepath.Join(audioDir, "old.opus")
	if err := os.WriteFile(p, make([]byte, 50), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := idx.record(search.Result{ID: "old"}, p, 50, "opus"); err != nil {
		t.Fatalf("record: %v", err)
	}

	// maxAge de 1ns ⇒ la entrada recién creada ya supera la antigüedad máxima.
	svc := New(repo, "", dir, 0, time.Nanosecond)
	if err := svc.Sweep(); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	if _, found, _ := idx.get("old"); found {
		t.Fatal("aged-out entry should have been swept")
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("aged-out file should have been deleted from disk")
	}
}

func TestClearEmptiesCache(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "vid1"})
	dir := t.TempDir()
	svc := New(repo, fakeYtDlp(t, "AUDIO"), dir, 0, 0)

	if _, err := svc.Download(context.Background(), search.Result{ID: "vid1"}); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if err := svc.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if _, ok := svc.Lookup("vid1"); ok {
		t.Fatal("Lookup after Clear = ok, want miss")
	}
	if _, err := os.Stat(filepath.Join(dir, "audio")); !os.IsNotExist(err) {
		t.Fatal("audio dir should be removed after Clear")
	}
}

// TestClearRemovesCoversDir verifica que Clear también purga la caché de
// portadas (covers/), que comparte el ciclo de vida de la caché de audio
// (task 3.5: la eviction de portadas se apoya en Evict/Clear).
func TestClearRemovesCoversDir(t *testing.T) {
	repo := newRepo(t)
	dir := t.TempDir()
	svc := New(repo, "", dir, 0, 0)

	coversDir := filepath.Join(dir, "covers")
	if err := os.MkdirAll(coversDir, 0o755); err != nil {
		t.Fatalf("mkdir covers: %v", err)
	}
	if err := os.WriteFile(filepath.Join(coversDir, "deadbeef.jpg"), []byte("img"), 0o644); err != nil {
		t.Fatalf("write cover: %v", err)
	}

	if err := svc.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(coversDir); !os.IsNotExist(err) {
		t.Fatal("covers dir should be removed after Clear")
	}
}

// TestEvictRemovesCoversDirWhenEntriesDropped verifica que cuando la expiración
// por antigüedad descarta entradas, Evict purga también covers/ para no dejar
// portadas huérfanas (task 3.5: piggyback en el ciclo de vida de la caché).
func TestEvictRemovesCoversDirWhenEntriesDropped(t *testing.T) {
	repo := newRepo(t, search.Result{ID: "old"})
	idx := newIndex(repo)
	dir := t.TempDir()
	audioDir := filepath.Join(dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatalf("mkdir audio: %v", err)
	}
	p := filepath.Join(audioDir, "old.opus")
	if err := os.WriteFile(p, make([]byte, 50), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}
	if err := idx.record(search.Result{ID: "old"}, p, 50, "opus"); err != nil {
		t.Fatalf("record: %v", err)
	}

	coversDir := filepath.Join(dir, "covers")
	if err := os.MkdirAll(coversDir, 0o755); err != nil {
		t.Fatalf("mkdir covers: %v", err)
	}
	if err := os.WriteFile(filepath.Join(coversDir, "deadbeef.miss"), nil, 0o644); err != nil {
		t.Fatalf("write miss: %v", err)
	}

	// maxAge de 1ns ⇒ la entrada se expira y se descarta, disparando la purga
	// de covers/.
	svc := New(repo, "", dir, 0, time.Nanosecond)
	if err := svc.Evict(); err != nil {
		t.Fatalf("Evict: %v", err)
	}
	if _, err := os.Stat(coversDir); !os.IsNotExist(err) {
		t.Fatal("covers dir should be removed after Evict drops an entry")
	}
}
