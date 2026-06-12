// Package lyrics obtiene la letra de la pista en reproducción desde una API
// comunitaria sin auth (lrclib), parsea letras sincronizadas (.lrc) y resuelve
// la línea activa según la posición de reproducción. Todo fallo o ausencia de
// letra se trata como "sin letra", sin bloquear la reproducción.
package lyrics

import (
	"sort"
	"strconv"
	"strings"
)

// Line es una línea de letra con su marca de tiempo en segundos. Para letras no
// sincronizadas, T es 0 y solo Text es significativo.
type Line struct {
	T    float64 // segundos desde el inicio de la pista
	Text string
}

// Lyrics es el resultado resuelto para una pista: cuando Synced es true, Lines
// contiene las líneas con marcas de tiempo ordenadas; en caso contrario Plain
// contiene el texto plano. Una Lyrics vacía representa "sin letra".
type Lyrics struct {
	Synced bool
	Lines  []Line
	Plain  string
}

// Empty indica si no hay letra disponible (ni sincronizada ni plana).
func (l Lyrics) Empty() bool {
	return !l.Synced && l.Plain == "" && len(l.Lines) == 0
}

// parseLRC parsea el cuerpo .lrc sincronizado. Cada línea con uno o más
// prefijos [mm:ss.xx] produce una Line por marca; las líneas sin marca válida
// se ignoran. El resultado queda ordenado por tiempo ascendente. Si no se
// encuentra ninguna marca, Synced es false.
func parseLRC(body string) Lyrics {
	var lines []Line
	for _, raw := range strings.Split(body, "\n") {
		stamps, text := parseLRCLine(raw)
		if len(stamps) == 0 {
			continue
		}
		for _, t := range stamps {
			lines = append(lines, Line{T: t, Text: text})
		}
	}
	if len(lines) == 0 {
		return Lyrics{}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].T < lines[j].T })
	return Lyrics{Synced: true, Lines: lines}
}

// parseLRCLine extrae las marcas de tiempo (en segundos) y el texto de una línea
// .lrc. Devuelve marcas vacías cuando la línea no empieza con una marca temporal
// válida (p. ej. etiquetas de metadatos como [ar:...] que no son mm:ss).
func parseLRCLine(raw string) (stamps []float64, text string) {
	rest := raw
	for {
		rest = strings.TrimLeft(rest, " \t")
		if !strings.HasPrefix(rest, "[") {
			break
		}
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			break
		}
		tag := rest[1:end]
		t, ok := parseTimestamp(tag)
		if !ok {
			// Etiqueta de metadatos, no temporal: detener el escaneo de marcas.
			break
		}
		stamps = append(stamps, t)
		rest = rest[end+1:]
	}
	return stamps, strings.TrimSpace(rest)
}

// parseTimestamp interpreta una marca "mm:ss", "mm:ss.xx" o "mm:ss:xx" en
// segundos. Devuelve ok=false si el formato no es temporal.
func parseTimestamp(s string) (float64, bool) {
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return 0, false
	}
	minPart := s[:colon]
	secPart := s[colon+1:]
	min, err := strconv.Atoi(strings.TrimSpace(minPart))
	if err != nil || min < 0 {
		return 0, false
	}
	// Algunos productores usan "mm:ss:cc" en vez de "mm:ss.cc".
	secPart = strings.Replace(secPart, ":", ".", 1)
	sec, err := strconv.ParseFloat(strings.TrimSpace(secPart), 64)
	if err != nil || sec < 0 {
		return 0, false
	}
	return float64(min)*60 + sec, true
}

// plainText construye una Lyrics no sincronizada a partir de texto plano. Una
// cadena vacía produce una Lyrics vacía ("sin letra").
func plainText(body string) Lyrics {
	body = strings.TrimSpace(body)
	if body == "" {
		return Lyrics{}
	}
	return Lyrics{Plain: body}
}

// LineAt devuelve el índice de la línea sincronizada activa para la posición
// sec (en segundos), es decir, la última línea cuyo tiempo es <= sec. Devuelve
// -1 cuando la letra no está sincronizada, está vacía o la posición es anterior
// a la primera marca. Usa búsqueda binaria, por lo que es estable ante saltos
// (seek) y avance normal por igual.
func (l Lyrics) LineAt(sec float64) int {
	if !l.Synced || len(l.Lines) == 0 {
		return -1
	}
	// sort.Search encuentra el primer índice con T > sec; la línea activa es el
	// anterior.
	i := sort.Search(len(l.Lines), func(i int) bool { return l.Lines[i].T > sec })
	return i - 1
}
