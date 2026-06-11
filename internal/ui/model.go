// Package ui implementa la interfaz de terminal de TerminalTube con Bubble Tea.
package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/alexcasdev/terminaltube/internal/config"
	"github.com/alexcasdev/terminaltube/internal/favorites"
	"github.com/alexcasdev/terminaltube/internal/history"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/playlist"
	"github.com/alexcasdev/terminaltube/internal/queue"
	"github.com/alexcasdev/terminaltube/internal/search"
	"github.com/alexcasdev/terminaltube/internal/storage"
)

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

	width, height int
	quitting      bool
}

// New construye el modelo inicial.
func New(cfg config.Config, s search.Searcher, p player.Player, h *history.History, pl *playlist.Service, fav *favorites.Service, logger *zap.Logger) Model {
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
		logger:    logger,
		keys:      defaultKeys(),
		styles:    defaultStyles(),
		input:     in,
		picker:    picker,
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
