// Package metadata provee un normalizador puro y determinista que deriva un par
// (artist, title) limpio a partir del (Title, Uploader) crudo de YouTube. Se usa
// únicamente como entrada para consultas salientes (letra/portada): nunca muta
// los datos almacenados ni realiza I/O.
package metadata

import (
	"regexp"
	"strings"

	"github.com/alexcasdev/terminaltube/internal/search"
)

// separators son los separadores artista/título reconocidos: guion ASCII " - "
// y guion largo " – " (en-dash). Se prueba el guion ASCII primero.
var separators = []string{" - ", " – "}

// tagPattern elimina etiquetas entre paréntesis o corchetes de ruido habitual en
// títulos de YouTube: (Official Video), (Official Music Video), [MV], (Lyrics),
// (Lyric Video), (Audio), (Visualizer), (HD)/(HQ)/(4K) y etiquetas de año
// (cuatro dígitos). La comparación es insensible a mayúsculas.
var tagPattern = regexp.MustCompile(`(?i)\s*[\(\[]\s*(?:` +
	`official(?:\s+(?:music\s+)?video|\s+audio|\s+lyric(?:s)?(?:\s+video)?)?|` +
	`music\s+video|` +
	`lyrics?(?:\s+video)?|` +
	`lyric\s+video|` +
	`audio|` +
	`visuali[sz]er|` +
	`m/?v|` +
	`hd|hq|4k|1080p|720p|` +
	`\d{4}` +
	`)\s*[\)\]]`)

// featPattern descarta los segmentos feat./ft. (y "featuring") y todo lo que les
// sigue, ya que confunden las búsquedas de letra/portada. Insensible a
// mayúsculas; tolera el segmento envuelto en paréntesis/corchetes.
var featPattern = regexp.MustCompile(`(?i)\s*[\(\[]?\s*\b(?:feat|ft|featuring)\b\.?.*$`)

// channelNoise elimina el ruido de nombre de canal al derivar el artista desde
// el uploader: sufijo "VEVO", el marcador "- Topic" de los canales autogenerados
// y la palabra "Official". Insensible a mayúsculas.
var (
	vevoSuffix    = regexp.MustCompile(`(?i)\s*vevo\s*$`)
	topicSuffix   = regexp.MustCompile(`(?i)\s*-\s*topic\s*$`)
	officialNoise = regexp.MustCompile(`(?i)\bofficial\b`)
)

// wsPattern colapsa cualquier secuencia de espacios en blanco a un solo espacio.
var wsPattern = regexp.MustCompile(`\s+`)

// Normalize deriva un par (artist, title) limpio a partir de r para usarlo como
// entrada de consulta. Es pura y determinista: no realiza I/O ni muta r.
//
// Si el título contiene un separador (" - "/" – "), el texto previo al primero
// se toma como artista y el resto como título. En ausencia de separador, el
// artista se deriva del Uploader (quitando VEVO, "- Topic" y "Official") y el
// título es el título completo. En ambos casos se eliminan etiquetas de ruido y
// segmentos feat./ft., y se colapsan los espacios.
func Normalize(r search.Result) (artist, title string) {
	rawTitle := strings.TrimSpace(r.Title)

	if a, t, ok := splitArtistTitle(rawTitle); ok {
		return cleanField(a), cleanTitle(t)
	}

	return deriveArtist(r.Uploader), cleanTitle(rawTitle)
}

// splitArtistTitle separa el título en (artista, título) por el primer separador
// reconocido. Devuelve ok=false cuando no hay separador.
func splitArtistTitle(title string) (artist, rest string, ok bool) {
	best := -1
	for _, sep := range separators {
		if i := strings.Index(title, sep); i >= 0 && (best < 0 || i < best) {
			best = i
			artist = title[:i]
			rest = title[i+len(sep):]
		}
	}
	if best < 0 {
		return "", "", false
	}
	return artist, rest, true
}

// cleanTitle limpia un título: descarta feat./ft., elimina etiquetas de ruido y
// colapsa espacios.
func cleanTitle(s string) string {
	s = featPattern.ReplaceAllString(s, "")
	s = tagPattern.ReplaceAllString(s, "")
	return collapse(s)
}

// cleanField limpia un campo genérico (p. ej. el artista tras un split):
// descarta feat./ft., elimina etiquetas y colapsa espacios.
func cleanField(s string) string {
	s = featPattern.ReplaceAllString(s, "")
	s = tagPattern.ReplaceAllString(s, "")
	return collapse(s)
}

// deriveArtist obtiene el artista a partir del nombre del canal (uploader),
// quitando el ruido VEVO, "- Topic" y "Official". Devuelve "" si el uploader es
// vacío.
func deriveArtist(uploader string) string {
	s := strings.TrimSpace(uploader)
	if s == "" {
		return ""
	}
	s = topicSuffix.ReplaceAllString(s, "")
	s = vevoSuffix.ReplaceAllString(s, "")
	s = officialNoise.ReplaceAllString(s, "")
	return collapse(s)
}

// collapse normaliza los espacios: colapsa secuencias internas a uno solo y
// recorta los extremos.
func collapse(s string) string {
	return strings.TrimSpace(wsPattern.ReplaceAllString(s, " "))
}
