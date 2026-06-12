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
type lyricsService interface {
	Fetch(ctx context.Context, track search.Result, queryTitle, queryArtist string) (lyrics.Lyrics, error)
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
	cfg       config.Config
	searcher  search.Searcher
	player    player.Player
	queue     *queue.Queue
	history   *history.History
	playlists *playlist.Service
	favorites *favorites.Service
	logger    *zap.Logger

	// Servicios de enriquecimiento (Fase 3). nil ⇒ feature apagada.
	cache    cacheService
	lyrics   lyricsService
	artwork  artworkService
	presence presenceService

	keys   keyMap
	styles styles
	input  textinput.Model
	picker list.Model

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

	// Estado de los paneles de enriquecimiento (Fase 3). Indexados por video id
	// para descartar respuestas que llegan tras un cambio de pista.
	curTrackID string          // id de la pista en reproducción
	curLyrics  lyrics.Lyrics   // letra de la pista actual
	curArtwork string          // portada renderizada de la pista actual
	lyricLine  int             // índice de la línea de letra resaltada (-1 = ninguna)
	cachedIDs  map[string]bool // ids con archivo local en caché (indicador)

	width, height int
	quitting      bool
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

	return Model{
		cfg:       cfg,
		searcher:  s,
		player:    p,
		queue:     queue.New(),
		history:   h,
		playlists: pl,
		favorites: fav,
		cache:     svc.Cache,
		lyrics:    svc.Lyrics,
		artwork:   svc.Artwork,
		presence:  svc.Presence,
		logger:    logger,
		keys:      defaultKeys(),
		styles:    defaultStyles(),
		input:     in,
		picker:    picker,
		lyricLine: -1,
		cachedIDs: make(map[string]bool),
		status:    "Pulsa / para buscar · L biblioteca.",
	}
}

// playlistItem adapta storage.Playlist al list.Item del picker de playlists.
type playlistItem struct{ pl storage.Playlist }

func (i playlistItem) Title() string       { return i.pl.Name }
func (i playlistItem) Description() string { return "" }
func (i playlistItem) FilterValue() string { return i.pl.Name }

// Init arranca el bucle de eventos del reproductor y el tick de progreso.
func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForEventCmd(m.player), tickCmd())
}
