// Este archivo resuelve la portada real (cover) de una pista a partir de su
// (artist, title) normalizado: consulta MusicBrainz para obtener una release
// MBID y descarga la portada frontal desde Cover Art Archive. El resolutor
// nunca interrumpe la reproducción: ante un miss, error de red o estado offline
// devuelve ok=false. Los resultados se cachean en disco (positivos y
// negativos) y las peticiones se limitan a ~1 req/s por etiqueta de MusicBrainz.
package artwork

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// userAgent identifica a la aplicación ante MusicBrainz/Cover Art Archive, como
// exige la etiqueta de uso de MusicBrainz. Reusa la cadena del servicio de
// letras para mantener una sola identidad de la aplicación.
const userAgent = "terminaltube (https://github.com/alexcasdev/terminaltube)"

// durationToleranceSec es la tolerancia fija (±7 s) del contraste de duración:
// una release de MusicBrainz solo se acepta si la longitud de su recording está
// dentro de este margen respecto a la duración de la pista. Si MusicBrainz no
// reporta longitud, se acepta el resultado mejor rankeado.
const durationToleranceSec = 7

// rateLimitInterval es el intervalo mínimo entre peticiones salientes (~1 req/s)
// para respetar la etiqueta de uso de MusicBrainz.
const rateLimitInterval = time.Second

// CoverResolver resuelve la ruta local de la portada real de una pista. La
// implementación es tolerante a fallos: devuelve ok=false ante un miss, estado
// offline o cualquier error, y nunca propaga un error que interrumpa la
// reproducción.
type CoverResolver interface {
	// Resolve devuelve la ruta local de la portada para el (artist, title)
	// normalizado, u ok=false si no hay coincidencia o no hay red.
	Resolve(ctx context.Context, artist, title string, durationSec int) (path string, ok bool)
}

// limiter limita la tasa de peticiones a ~1 req/s usando un time.Ticker y un
// mutex de stdlib (sin dependencias externas). wait bloquea hasta que el ticker
// habilita la siguiente petición o el contexto se cancela.
type limiter struct {
	mu     sync.Mutex
	ticker *time.Ticker
}

// newLimiter construye un limitador con el intervalo dado.
func newLimiter(interval time.Duration) *limiter {
	return &limiter{ticker: time.NewTicker(interval)}
}

// wait bloquea hasta el siguiente tick del limitador o hasta que ctx se cancele.
// Serializa a los llamadores con el mutex para que cada uno consuma un tick.
func (l *limiter) wait(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.ticker.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// mbCoverResolver implementa CoverResolver contra MusicBrainz + Cover Art
// Archive usando solo net/http de la stdlib.
type mbCoverResolver struct {
	client     *http.Client
	limiter    *limiter
	cacheDir   string // directorio covers/ bajo el XDG cache home
	mbBaseURL  string // base de MusicBrainz (override en tests)
	caaBaseURL string // base de Cover Art Archive (override en tests)
}

// NewCoverResolver construye el resolutor de portadas. cacheHome es el XDG cache
// home de la aplicación: las portadas se guardan bajo cacheHome/covers/. client
// es opcional (nil ⇒ uno con timeout por defecto).
func NewCoverResolver(cacheHome string, client *http.Client) CoverResolver {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &mbCoverResolver{
		client:     client,
		limiter:    newLimiter(rateLimitInterval),
		cacheDir:   filepath.Join(cacheHome, "covers"),
		mbBaseURL:  "https://musicbrainz.org",
		caaBaseURL: "https://coverartarchive.org",
	}
}

// mbRecordingSearch es la forma de la respuesta de búsqueda de recordings de
// MusicBrainz. Solo se interpretan los campos necesarios para elegir release y
// contrastar duración.
type mbRecordingSearch struct {
	Recordings []mbRecording `json:"recordings"`
}

type mbRecording struct {
	// Length es la duración del recording en milisegundos (puede faltar).
	Length   int         `json:"length"`
	Releases []mbRelease `json:"releases"`
}

type mbRelease struct {
	ID string `json:"id"`
}

// cacheKey deriva una clave content-addressed estable a partir del (artist,
// title) normalizado: sha1(artist|title) en minúsculas. Dedup entre videoIDs
// distintos con la misma pista.
func cacheKey(artist, title string) string {
	sum := sha1.Sum([]byte(strings.ToLower(artist) + "|" + strings.ToLower(title)))
	return hex.EncodeToString(sum[:])
}

// missPath devuelve la ruta del marcador negativo (.miss) para una clave.
func (r *mbCoverResolver) missPath(key string) string {
	return filepath.Join(r.cacheDir, key+".miss")
}

// cachedImage busca una imagen ya cacheada para la clave (cualquier extensión
// soportada). Devuelve la ruta y ok=true si existe un archivo no vacío.
func (r *mbCoverResolver) cachedImage(key string) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(r.cacheDir, key+".*"))
	if err != nil {
		return "", false
	}
	for _, m := range matches {
		if strings.HasSuffix(m, ".miss") {
			continue
		}
		if info, statErr := os.Stat(m); statErr == nil && !info.IsDir() && info.Size() > 0 {
			return m, true
		}
	}
	return "", false
}

// Resolve implementa CoverResolver. Orden: acierto en caché (imagen o .miss) ⇒
// resultado cacheado sin red; en otro caso consulta MusicBrainz → Cover Art
// Archive. Solo escribe el marcador .miss ante un negativo DEFINITIVO (sin
// release usable en MusicBrainz, o 404 / sin portada frontal en Cover Art
// Archive). Ante un fallo TRANSITORIO de transporte (error de red, timeout, 5xx,
// error de lectura) devuelve ok=false SIN escribir .miss, para reintentar en la
// próxima reproducción y no envenenar la caché por un corte momentáneo.
func (r *mbCoverResolver) Resolve(ctx context.Context, artist, title string, durationSec int) (string, bool) {
	if title == "" {
		return "", false
	}
	key := cacheKey(artist, title)

	// Caché primero: imagen positiva o marcador negativo evitan toda petición.
	if path, ok := r.cachedImage(key); ok {
		return path, true
	}
	if _, err := os.Stat(r.missPath(key)); err == nil {
		return "", false
	}

	mbid, outcome := r.lookupRelease(ctx, artist, title, durationSec)
	switch outcome {
	case lookupTransient:
		// Fallo transitorio: no cachear, reintentar la próxima vez.
		return "", false
	case lookupNoMatch:
		// Negativo definitivo: MusicBrainz no tiene release usable.
		r.writeMiss(key)
		return "", false
	}

	path, outcome := r.fetchCover(ctx, mbid, key)
	switch outcome {
	case lookupTransient:
		return "", false
	case lookupNoMatch:
		// Negativo definitivo: Cover Art Archive no tiene portada frontal (404).
		r.writeMiss(key)
		return "", false
	}
	return path, true
}

// coverOutcome clasifica el desenlace de una consulta saliente para decidir si
// cachear un negativo. lookupFound = éxito; lookupNoMatch = negativo definitivo
// (cacheable con .miss); lookupTransient = fallo de transporte recuperable (NO
// cacheable, se reintenta).
type coverOutcome int

const (
	lookupFound coverOutcome = iota
	lookupNoMatch
	lookupTransient
)

// lookupRelease consulta la búsqueda de recordings de MusicBrainz y devuelve la
// release MBID del mejor candidato que pase el contraste de duración. El segundo
// valor clasifica el desenlace: lookupFound con MBID; lookupNoMatch (negativo
// definitivo) cuando una respuesta 200 válida no contiene release usable;
// lookupTransient ante cualquier fallo de transporte recuperable (error de red,
// timeout, código != 200, error de lectura o JSON ilegible).
func (r *mbCoverResolver) lookupRelease(ctx context.Context, artist, title string, durationSec int) (string, coverOutcome) {
	if err := r.limiter.wait(ctx); err != nil {
		return "", lookupTransient
	}

	query := `recording:"` + escapeLucene(title) + `"`
	if artist != "" {
		query += ` AND artist:"` + escapeLucene(artist) + `"`
	}
	endpoint := r.mbBaseURL + "/ws/2/recording?fmt=json&limit=5&query=" + queryEscape(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", lookupTransient
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", lookupTransient
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Un código != 200 (5xx, 429, etc.) es un fallo transitorio: no se
		// cachea negativo, se reintenta más tarde.
		return "", lookupTransient
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", lookupTransient
	}

	var search mbRecordingSearch
	if err := json.Unmarshal(data, &search); err != nil {
		return "", lookupTransient
	}

	// Los resultados vienen ordenados por score: el primero es el mejor
	// rankeado. Se acepta la primera release de un recording cuya longitud pase
	// el contraste de duración; si la longitud falta se acepta igualmente.
	for i := range search.Recordings {
		rec := &search.Recordings[i]
		if !durationMatches(rec.Length, durationSec) {
			continue
		}
		for _, rel := range rec.Releases {
			if rel.ID != "" {
				return rel.ID, lookupFound
			}
		}
	}
	// Respuesta 200 válida sin release usable: negativo definitivo.
	return "", lookupNoMatch
}

// durationMatches aplica el contraste de duración ±7 s. lengthMs es la longitud
// del recording de MusicBrainz en milisegundos; si es <=0 (MusicBrainz no la
// reporta) se acepta el resultado. durationSec <=0 (pista sin duración) también
// acepta.
func durationMatches(lengthMs, durationSec int) bool {
	if lengthMs <= 0 || durationSec <= 0 {
		return true
	}
	diff := lengthMs/1000 - durationSec
	if diff < 0 {
		diff = -diff
	}
	return diff <= durationToleranceSec
}

// fetchCover descarga la portada frontal /front-500 de Cover Art Archive para la
// release MBID y la guarda en covers/<key>.<ext>. El segundo valor clasifica el
// desenlace: lookupFound con la ruta escrita; lookupNoMatch (negativo
// definitivo) cuando CAA responde 404 (la release no tiene portada frontal);
// lookupTransient ante cualquier otro fallo recuperable (error de red, timeout,
// 5xx, cuerpo vacío, error de lectura o de escritura en disco).
func (r *mbCoverResolver) fetchCover(ctx context.Context, mbid, key string) (string, coverOutcome) {
	if err := r.limiter.wait(ctx); err != nil {
		return "", lookupTransient
	}

	endpoint := r.caaBaseURL + "/release/" + mbid + "/front-500"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", lookupTransient
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", lookupTransient
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		// La release no tiene portada frontal: negativo definitivo, cacheable.
		return "", lookupNoMatch
	}
	if resp.StatusCode != http.StatusOK {
		// 5xx u otros códigos: fallo transitorio, no se cachea negativo.
		return "", lookupTransient
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || len(data) == 0 {
		return "", lookupTransient
	}

	ext := imageExt(resp.Header.Get("Content-Type"))
	if err := os.MkdirAll(r.cacheDir, 0o755); err != nil {
		return "", lookupTransient
	}
	path := filepath.Join(r.cacheDir, key+"."+ext)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", lookupTransient
	}
	return path, lookupFound
}

// writeMiss escribe el marcador negativo vacío covers/<key>.miss para cachear un
// no-match y evitar reconsultar la red en futuras reproducciones de la pista.
func (r *mbCoverResolver) writeMiss(key string) {
	if err := os.MkdirAll(r.cacheDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(r.missPath(key), nil, 0o644)
}

// imageExt deriva la extensión de archivo a partir del Content-Type de la
// respuesta de Cover Art Archive. Por defecto "jpg" (la portada habitual).
func imageExt(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(ct, "png"):
		return "png"
	case strings.Contains(ct, "webp"):
		return "webp"
	case strings.Contains(ct, "gif"):
		return "gif"
	default:
		return "jpg"
	}
}

// escapeLucene escapa los caracteres especiales de la sintaxis Lucene usada por
// la búsqueda de MusicBrainz, evitando que comillas u operadores en el texto
// rompan la consulta.
func escapeLucene(s string) string {
	const special = `+-&|!(){}[]^"~*?:\/`
	var b strings.Builder
	for _, c := range s {
		if strings.ContainsRune(special, c) {
			b.WriteByte('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// queryEscape codifica el valor del parámetro query para la URL. Se usa una
// codificación manual porque url.QueryEscape convierte el espacio en '+', que
// MusicBrainz interpreta de forma distinta; %20 es inequívoco.
func queryEscape(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if isUnreserved(c) {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			const hexd = "0123456789ABCDEF"
			b.WriteByte(hexd[c>>4])
			b.WriteByte(hexd[c&0x0f])
		}
	}
	return b.String()
}

// isUnreserved indica si un byte puede ir sin codificar en una query URL
// (caracteres no reservados de RFC 3986).
func isUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-' || c == '_' || c == '.' || c == '~':
		return true
	default:
		return false
	}
}
