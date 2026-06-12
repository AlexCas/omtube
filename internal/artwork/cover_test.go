package artwork

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestResolver construye un mbCoverResolver apuntado a servidores de prueba,
// con un directorio covers/ temporal y un limitador rápido para no ralentizar
// los tests.
func newTestResolver(t *testing.T, mbURL, caaURL string) *mbCoverResolver {
	t.Helper()
	return &mbCoverResolver{
		client:     &http.Client{Timeout: 5 * time.Second},
		limiter:    newLimiter(time.Millisecond),
		cacheDir:   filepath.Join(t.TempDir(), "covers"),
		mbBaseURL:  mbURL,
		caaBaseURL: caaURL,
	}
}

// mbResponse es una respuesta de búsqueda de recordings con una release y
// longitud opcional (ms). length<=0 omite el campo.
func mbResponse(mbid string, lengthMs int) string {
	lengthField := ""
	if lengthMs > 0 {
		lengthField = `"length":` + itoa(lengthMs) + `,`
	}
	return `{"recordings":[{` + lengthField + `"releases":[{"id":"` + mbid + `"}]}]}`
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestCoverResolve(t *testing.T) {
	const mbid = "11111111-1111-1111-1111-111111111111"

	t.Run("real cover resolved", func(t *testing.T) {
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(mbResponse(mbid, 185000)))
		}))
		defer mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/front-500") {
				t.Errorf("CAA: ruta inesperada %q", r.URL.Path)
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("\xff\xd8\xff fake jpeg bytes"))
		}))
		defer caa.Close()

		res := newTestResolver(t, mb.URL, caa.URL)
		path, ok := res.Resolve(context.Background(), "Linkin Park", "Numb", 185)
		if !ok {
			t.Fatal("se esperaba una portada resuelta")
		}
		if !strings.HasSuffix(path, ".jpg") {
			t.Errorf("extensión inesperada: %q", path)
		}
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Errorf("la portada no se escribió en disco: %v", err)
		}
	})

	t.Run("cached lookup avoids second request", func(t *testing.T) {
		var mbHits int32
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&mbHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(mbResponse(mbid, 0)))
		}))
		defer mb.Close()
		var caaHits int32
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&caaHits, 1)
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("\x89PNG fake png"))
		}))
		defer caa.Close()

		res := newTestResolver(t, mb.URL, caa.URL)
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 0); !ok {
			t.Fatal("primer Resolve debió acertar")
		}
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 0); !ok {
			t.Fatal("segundo Resolve debió acertar desde caché")
		}
		if got := atomic.LoadInt32(&mbHits); got != 1 {
			t.Errorf("MusicBrainz consultado %d veces; se esperaba 1 (caché)", got)
		}
		if got := atomic.LoadInt32(&caaHits); got != 1 {
			t.Errorf("Cover Art Archive consultado %d veces; se esperaba 1 (caché)", got)
		}
	})

	t.Run("negative result writes .miss and is cached", func(t *testing.T) {
		var mbHits int32
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&mbHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(mbResponse(mbid, 200000)))
		}))
		defer mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r) // CAA sin portada para la release.
		}))
		defer caa.Close()

		res := newTestResolver(t, mb.URL, caa.URL)
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 200); ok {
			t.Fatal("404 de CAA debió producir ok=false")
		}
		miss := res.missPath(cacheKey("Artist", "Song"))
		if _, err := os.Stat(miss); err != nil {
			t.Errorf("no se escribió el marcador .miss: %v", err)
		}
		// Segundo Resolve: el .miss cacheado evita nuevas peticiones.
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 200); ok {
			t.Fatal("segundo Resolve debió seguir siendo negativo")
		}
		if got := atomic.LoadInt32(&mbHits); got != 1 {
			t.Errorf("MusicBrainz consultado %d veces; se esperaba 1 (.miss cacheado)", got)
		}
	})

	t.Run("transient MusicBrainz 5xx does not write .miss", func(t *testing.T) {
		// Un 5xx de MusicBrainz es un fallo transitorio: no debe envenenar la
		// caché con .miss, para reintentar en la próxima reproducción.
		var mbHits int32
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&mbHits, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("CAA no debió consultarse tras un fallo de MusicBrainz")
		}))
		defer caa.Close()

		res := newTestResolver(t, mb.URL, caa.URL)
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 0); ok {
			t.Fatal("un 5xx de MusicBrainz debió producir ok=false")
		}
		miss := res.missPath(cacheKey("Artist", "Song"))
		if _, err := os.Stat(miss); err == nil {
			t.Fatal("un fallo transitorio NO debe escribir el marcador .miss")
		}
		// Segundo Resolve: al no haber .miss, vuelve a consultar MusicBrainz.
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 0); ok {
			t.Fatal("segundo Resolve debió seguir siendo negativo")
		}
		if got := atomic.LoadInt32(&mbHits); got != 2 {
			t.Errorf("MusicBrainz consultado %d veces; se esperaba 2 (sin .miss, se reintenta)", got)
		}
	})

	t.Run("transient CAA 5xx does not write .miss", func(t *testing.T) {
		// MusicBrainz resuelve una release, pero Cover Art Archive falla con 5xx
		// (transitorio): no debe escribirse .miss.
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(mbResponse(mbid, 0)))
		}))
		defer mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer caa.Close()

		res := newTestResolver(t, mb.URL, caa.URL)
		if _, ok := res.Resolve(context.Background(), "Artist", "SongCAA", 0); ok {
			t.Fatal("un 5xx de CAA debió producir ok=false")
		}
		miss := res.missPath(cacheKey("Artist", "SongCAA"))
		if _, err := os.Stat(miss); err == nil {
			t.Fatal("un fallo transitorio de CAA NO debe escribir el marcador .miss")
		}
	})

	t.Run("transient transport error does not write .miss", func(t *testing.T) {
		// Servidor de MusicBrainz cerrado de inmediato: el dial falla (error de
		// transporte transitorio). No debe escribirse .miss.
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		mbURL := mb.URL
		mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer caa.Close()

		res := newTestResolver(t, mbURL, caa.URL)
		if _, ok := res.Resolve(context.Background(), "Artist", "SongNet", 0); ok {
			t.Fatal("un error de transporte debió producir ok=false")
		}
		miss := res.missPath(cacheKey("Artist", "SongNet"))
		if _, err := os.Stat(miss); err == nil {
			t.Fatal("un error de transporte NO debe escribir el marcador .miss")
		}
	})

	t.Run("duration mismatch rejected", func(t *testing.T) {
		var caaHits int32
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Recording de 300 s frente a una pista de 185 s: fuera de ±7 s.
			_, _ = w.Write([]byte(mbResponse(mbid, 300000)))
		}))
		defer mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&caaHits, 1)
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("jpeg"))
		}))
		defer caa.Close()

		res := newTestResolver(t, mb.URL, caa.URL)
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 185); ok {
			t.Fatal("la diferencia de duración debió rechazar la release")
		}
		if got := atomic.LoadInt32(&caaHits); got != 0 {
			t.Errorf("no debió consultarse Cover Art Archive; hits=%d", got)
		}
	})

	t.Run("offline yields ok false", func(t *testing.T) {
		// Servidores cerrados de inmediato: cualquier dial falla.
		mb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		mbURL := mb.URL
		mb.Close()
		caa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		caaURL := caa.URL
		caa.Close()

		res := newTestResolver(t, mbURL, caaURL)
		if _, ok := res.Resolve(context.Background(), "Artist", "Song", 185); ok {
			t.Fatal("estado offline debió producir ok=false")
		}
	})
}

func TestDurationMatches(t *testing.T) {
	tests := []struct {
		name     string
		lengthMs int
		durSec   int
		want     bool
	}{
		{"exact match", 185000, 185, true},
		{"within tolerance high", 192000, 185, true},
		{"within tolerance low", 178000, 185, true},
		{"just outside tolerance", 193000, 185, false},
		{"mb omits length accepts", 0, 185, true},
		{"track without duration accepts", 185000, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := durationMatches(tt.lengthMs, tt.durSec); got != tt.want {
				t.Errorf("durationMatches(%d, %d) = %v; want %v", tt.lengthMs, tt.durSec, got, tt.want)
			}
		})
	}
}
