// Package ui implementa la interfaz de terminal de Omusic con Bubble Tea.
package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/alexcasdev/terminaltube/internal/config"
	"github.com/alexcasdev/terminaltube/internal/favorites"
	"github.com/alexcasdev/terminaltube/internal/history"
	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/playlist"
	"github.com/alexcasdev/terminaltube/internal/queue"
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

// Servicios de enriquecimiento desacoplados tras interfaces para poder dejarlos
// en nil cuando el toggle está apagado (degradación a la conducta de la Fase 2)
// e inyectar dobles en tests.

// cacheService resuelve y descarga archivos de audio cacheados. Un valor nil
// equivale a "caché desactivada".
type cacheService interface {
	Lookup(id string) (string, bool)
	Download(ctx context.Context, r search.Result) (string, error)
}

// lyricsService resuelve la letra de una pista. Un valor nil ⇒ panel apagado.
// Search/SelectCandidate dan la búsqueda manual: el usuario teclea una consulta,
// elige un candidato y la referencia queda guardada para reusarse al reproducir.
type lyricsService interface {
	Fetch(ctx context.Context, track search.Result, queryTitle, queryArtist string) (lyrics.Lyrics, error)
	Search(ctx context.Context, query string) ([]lyrics.Candidate, error)
	SelectCandidate(ctx context.Context, track search.Result, c lyrics.Candidate) (lyrics.Lyrics, error)
}

// artworkService renderiza la portada de una pista. Un valor nil ⇒ panel apagado.
type artworkService interface {
	Render(ctx context.Context, track search.Result, w, h int) string
}

// presenceService publica la pista actual como presencia de Discord. Un valor
// nil ⇒ presencia desactivada.
type presenceService interface {
	Set(title, artist string)
	Clear()
}

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeLibrary
	modePicker         // selección de playlist (add-to-playlist) sobre modeLibrary
	modeCreatePlaylist // prompt de nombre para crear playlist sobre modeLibrary
	modeURLInput       // prompt para pegar una URL de vídeo de YouTube
	modeImportURL      // prompt para pegar una URL de playlist de YouTube
	modeImportName     // prompt de nombre tras resolver una playlist importada
	modeLyricsSearch   // prompt de consulta para la búsqueda manual de letra
	modeLyricsPicker   // selección de candidato de letra
	modeResults        // modal de pantalla completa con los resultados de búsqueda
)

// librarySection identifica la sección activa del modo biblioteca.
type librarySection int

const (
	sectionPlaylists librarySection = iota
	sectionFavorites
	sectionHistory
	librarySectionCount
)

// Model es el estado de la TUI.
type Model struct {
	cfg        config.Config
	searcher   search.Searcher
	resolver   search.Resolver         // resuelve URLs de vídeo; nil ⇒ no soportado
	plResolver search.PlaylistResolver // resuelve URLs de playlist; nil ⇒ no soportado
	player     player.Player
	queue      *queue.Queue
	history    *history.History
	playlists  *playlist.Service
	favorites  *favorites.Service
	logger     *zap.Logger

	// Servicios de enriquecimiento (Fase 3). nil ⇒ feature apagada.
	cache    cacheService
	lyrics   lyricsService
	artwork  artworkService
	presence presenceService

	keys        keyMap
	styles      styles
	input       textinput.Model
	picker      list.Model
	resultsList list.Model // modal de resultados de búsqueda (modeResults)

	mode      mode
	results   []search.Result
	cursor    int
	status    string
	searching bool
	started   bool // ya se inició la reproducción al menos una vez
	pos, dur  float64

	// Estado del modo biblioteca.
	libSection   librarySection
	libCursor    int
	libPlaylists []storage.Playlist
	libFavorites []search.Result
	libHistory   []search.Result

	// Estado del picker de add-to-playlist.
	pickerTrack  search.Result
	pickerReturn mode // modo al que volver tras cerrar el picker

	// Estado de los flujos por URL / importación / búsqueda manual de letra.
	importTracks []search.Result    // pistas resueltas de una playlist, a la espera de nombre
	importTitle  string             // título de la playlist de YouTube importada (informativo)
	lyricsTrack  search.Result      // pista objetivo de la búsqueda manual de letra
	lyricCands   []lyrics.Candidate // candidatos de la última búsqueda manual

	// Estado de los paneles de enriquecimiento (Fase 3). Indexados por video id
	// para descartar respuestas que llegan tras un cambio de pista.
	curTrackID string          // id de la pista en reproducción
	curLyrics  lyrics.Lyrics   // letra de la pista actual
	curArtwork string          // portada renderizada de la pista actual
	lyricLine  int             // índice de la línea de letra resaltada (-1 = ninguna)
	cachedIDs  map[string]bool // ids con archivo local en caché (indicador)

	width, height int
	quitting      bool

	// animFrame avanza con el tick de animación mientras hay reproducción y
	// alimenta el visualizador de barras bajo las instrucciones. En pausa o sin
	// pista no avanza, de modo que las barras quedan planas.
	animFrame int
}

// Services agrupa los servicios de enriquecimiento opcionales (Fase 3). Cualquier
// campo nil deja su feature apagada: con todos en nil la UI se comporta
// exactamente como en la Fase 2.
type Services struct {
	Cache    cacheService
	Lyrics   lyricsService
	Artwork  artworkService
	Presence presenceService
}

// New construye el modelo inicial. svc agrupa los servicios de enriquecimiento;
// el cero-value (todos nil) reproduce la conducta de la Fase 2.
func New(cfg config.Config, s search.Searcher, p player.Player, h *history.History, pl *playlist.Service, fav *favorites.Service, svc Services, logger *zap.Logger) Model {
	in := textinput.New()
	in.Placeholder = "Buscar canción…"
	in.Prompt = "🔎 "
	in.CharLimit = 200

	picker := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	picker.Title = "Añadir a playlist"
	picker.SetShowHelp(false)
	picker.SetShowStatusBar(false)
	picker.SetFilteringEnabled(false)

	// resultsList es el modal de pantalla completa que presenta los resultados de
	// una búsqueda multi-resultado. Reutiliza el mismo patrón que picker.
	rl := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	rl.Title = "Resultados"
	rl.SetShowHelp(false)
	rl.SetShowStatusBar(false)
	rl.SetFilteringEnabled(false)
	// El binding de salida de bubbles/list está ligado a "q"/"esc" (Quit) y "ctrl+c"
	// (ForceQuit), y devuelve tea.Quit saltándose el cierre limpio del reproductor.
	// DisableQuitKeybindings fija el flag persistente que el list respeta (un
	// SetEnabled directo se re-habilita en su propio Update). Esc lo gestiona
	// updateResultsMode (cierra el modal) y ctrl+c se intercepta allí para un cierre
	// limpio; "q" queda inerte dentro del modal.
	rl.DisableQuitKeybindings()

	// El buscador concreto (yt-dlp) también resuelve URLs de vídeo y de playlist;
	// se exponen tras sus interfaces para que la UI pueda degradar si no están.
	resolver, _ := s.(search.Resolver)
	plResolver, _ := s.(search.PlaylistResolver)

	return Model{
		cfg:         cfg,
		searcher:    s,
		resolver:    resolver,
		plResolver:  plResolver,
		player:      p,
		queue:       queue.New(),
		history:     h,
		playlists:   pl,
		favorites:   fav,
		cache:       svc.Cache,
		lyrics:      svc.Lyrics,
		artwork:     svc.Artwork,
		presence:    svc.Presence,
		logger:      logger,
		keys:        defaultKeys(),
		styles:      defaultStyles(),
		input:       in,
		picker:      picker,
		resultsList: rl,
		lyricLine:   -1,
		cachedIDs:   make(map[string]bool),
		status:      "Pulsa / para buscar · L biblioteca.",
	}
}

// playlistItem adapta storage.Playlist al list.Item del picker de playlists.
type playlistItem struct{ pl storage.Playlist }

func (i playlistItem) Title() string       { return i.pl.Name }
func (i playlistItem) Description() string { return "" }
func (i playlistItem) FilterValue() string { return i.pl.Name }

// candidateItem adapta lyrics.Candidate al list.Item del picker de candidatos de
// letra en la búsqueda manual.
type candidateItem struct{ c lyrics.Candidate }

func (i candidateItem) Title() string {
	if i.c.Synced {
		return "🎵 " + i.c.Title
	}
	return i.c.Title
}
func (i candidateItem) Description() string { return i.c.Artist }
func (i candidateItem) FilterValue() string { return i.c.Title }

// resultItem adapta search.Result al list.Item del modal de resultados de búsqueda.
// El indicador de caché (mark) se antepone al título para no requerir un delegate
// personalizado, igual que candidateItem usa un prefijo emoji.
type resultItem struct {
	r    search.Result
	mark string
}

func (i resultItem) Title() string       { return i.mark + i.r.Title }
func (i resultItem) Description() string { return i.r.Uploader }
func (i resultItem) FilterValue() string { return i.r.Title }

// Init arranca el bucle de eventos del reproductor y el tick de progreso.
func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForEventCmd(m.player), tickCmd(), animTickCmd())
}
