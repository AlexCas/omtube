package search

import (
	"net/url"
	"strings"
)

// URLKind clasifica una URL de YouTube introducida por el usuario.
type URLKind int

const (
	// URLUnknown indica que la URL no es una URL de YouTube reconocible.
	URLUnknown URLKind = iota
	// URLVideo indica una URL de un vídeo único.
	URLVideo
	// URLPlaylist indica una URL de una playlist.
	URLPlaylist
)

// ClassifyURL clasifica una URL de YouTube y extrae el identificador relevante:
// el id de vídeo para URLVideo o el id de lista para URLPlaylist. Una URL de tipo
// `watch?v=...` se trata como vídeo aunque incluya un parámetro `list=`; solo
// `/playlist?list=...` (o una URL con `list=` sin vídeo) se trata como playlist.
func ClassifyURL(raw string) (URLKind, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return URLUnknown, ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return URLUnknown, ""
	}

	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	q := u.Query()

	firstSegment := func(s string) string {
		s = strings.Trim(s, "/")
		if i := strings.IndexByte(s, '/'); i >= 0 {
			s = s[:i]
		}
		return s
	}

	switch host {
	case "youtu.be":
		if id := firstSegment(u.Path); id != "" {
			return URLVideo, id
		}
		return URLUnknown, ""

	case "youtube.com", "m.youtube.com", "music.youtube.com":
		switch {
		case u.Path == "/playlist":
			if list := q.Get("list"); list != "" {
				return URLPlaylist, list
			}
		case u.Path == "/watch":
			if v := q.Get("v"); v != "" {
				return URLVideo, v // watch + list ⇒ vídeo
			}
			if list := q.Get("list"); list != "" {
				return URLPlaylist, list
			}
		case strings.HasPrefix(u.Path, "/shorts/"):
			if id := firstSegment(strings.TrimPrefix(u.Path, "/shorts/")); id != "" {
				return URLVideo, id
			}
		case strings.HasPrefix(u.Path, "/embed/"):
			if id := firstSegment(strings.TrimPrefix(u.Path, "/embed/")); id != "" {
				return URLVideo, id
			}
		}
		// Host de YouTube con ruta no reconocida pero con `list=`: tratar como playlist.
		if list := q.Get("list"); list != "" {
			return URLPlaylist, list
		}
		return URLUnknown, ""
	}

	return URLUnknown, ""
}

// videoURL reconstruye una URL canónica de vídeo a partir de un id.
func videoURL(id string) string { return "https://www.youtube.com/watch?v=" + id }

// playlistURL reconstruye una URL canónica de playlist a partir de un id de lista.
func playlistURL(id string) string { return "https://www.youtube.com/playlist?list=" + id }
