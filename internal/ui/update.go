package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/search"
)

// Update procesa mensajes y entrada de teclado.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.picker.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			return m.updateSearchMode(msg)
		case modeLibrary:
			return m.updateLibraryMode(msg)
		case modePicker:
			return m.updatePickerMode(msg)
		case modeCreatePlaylist:
			return m.updateCreatePlaylistMode(msg)
		default:
			return m.updateNormalMode(msg)
		}

	case searchResultsMsg:
		m.searching = false
		if msg.err != nil {
			m.status = m.styles.errorMsg.Render("Error de búsqueda: " + msg.err.Error())
			return m, nil
		}
		m.results = msg.results
		m.cursor = 0
		if len(m.results) == 0 {
			m.status = "Sin resultados."
		} else {
			m.status = fmt.Sprintf("%d resultados. Enter para encolar.", len(m.results))
		}
		return m, nil

	case loadedMsg:
		if msg.err != nil {
			m.status = m.styles.errorMsg.Render("No se pudo reproducir: " + msg.err.Error())
			return m, nil
		}
		if err := m.history.Add(msg.track); err != nil && m.logger != nil {
			m.logger.Warn("no se pudo guardar historial: " + err.Error())
		}
		m.status = "▶ " + msg.track.Title
		return m, nil

	case playerEventMsg:
		return m.handlePlayerEvent(msg)

	case tickMsg:
		// El sondeo de posición corre en su propio Cmd para no bloquear Update.
		return m, tea.Batch(fetchPositionCmd(m.player), tickCmd())

	case posMsg:
		m.pos, m.dur = msg.pos, msg.dur
		m.advanceLyric()
		return m, nil

	case lyricsMsg:
		// Descartar respuestas obsoletas de una pista ya cambiada.
		if msg.videoID == m.curTrackID {
			m.curLyrics = msg.lyrics
			m.lyricLine = -1
			m.advanceLyric()
		}
		return m, nil

	case artworkMsg:
		if msg.videoID == m.curTrackID {
			m.curArtwork = msg.art
		}
		return m, nil

	case cacheDoneMsg:
		if msg.err != nil {
			m.warn("no se pudo cachear la pista: " + msg.err.Error())
			return m, nil
		}
		if m.cachedIDs == nil {
			m.cachedIDs = make(map[string]bool)
		}
		m.cachedIDs[msg.videoID] = true
		return m, nil
	}

	return m, nil
}

// advanceLyric recalcula la línea de letra resaltada según la posición actual.
func (m *Model) advanceLyric() {
	if m.curLyrics.Synced {
		m.lyricLine = m.curLyrics.LineAt(m.pos)
	}
}

func (m Model) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = modeNormal
		m.input.Blur()
		return m, nil
	case key.Matches(msg, m.keys.Enqueue): // enter envía la búsqueda
		q := m.input.Value()
		m.mode = modeNormal
		m.input.Blur()
		if q == "" {
			return m, nil
		}
		m.searching = true
		m.status = "Buscando…"
		return m, doSearchCmd(m.searcher, q, m.cfg.SearchResults)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		_ = m.player.Close()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Search):
		m.mode = modeSearch
		m.input.SetValue("")
		m.input.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Enqueue):
		if m.cursor >= 0 && m.cursor < len(m.results) {
			track := m.results[m.cursor]
			m.queue.Add(track)
			if !m.started {
				m.started = true
				return m, loadTrackCmd(m.player, m.cache, track)
			}
			m.status = "Añadido a la cola: " + track.Title
		}
		return m, nil

	case key.Matches(msg, m.keys.Toggle):
		if _, ok := m.queue.Current(); !ok {
			return m, nil // nada en reproducción que pausar
		}
		if err := m.player.TogglePause(); err != nil {
			m.status = m.styles.errorMsg.Render(err.Error())
		}
		return m, nil

	case key.Matches(msg, m.keys.Next):
		if m.queue.Next() {
			if cur, ok := m.queue.Current(); ok {
				return m, loadTrackCmd(m.player, m.cache, cur)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Prev):
		if m.queue.Prev() {
			if cur, ok := m.queue.Current(); ok {
				return m, loadTrackCmd(m.player, m.cache, cur)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.VolUp):
		v, _ := m.player.AddVolume(5)
		m.status = fmt.Sprintf("Volumen: %d", v)
		return m, nil

	case key.Matches(msg, m.keys.VolDown):
		v, _ := m.player.AddVolume(-5)
		m.status = fmt.Sprintf("Volumen: %d", v)
		return m, nil

	case key.Matches(msg, m.keys.Library):
		m.openLibrary()
		return m, nil

	case key.Matches(msg, m.keys.Favorite):
		if track, ok := m.selectedResult(); ok {
			m.toggleFavorite(track)
		}
		return m, nil

	case key.Matches(msg, m.keys.AddToPlaylist):
		if track, ok := m.selectedResult(); ok {
			return m.openPlaylistPicker(track)
		}
		return m, nil
	}
	return m, nil
}

// selectedResult devuelve la pista bajo el cursor en la lista de resultados.
func (m Model) selectedResult() (search.Result, bool) {
	if m.cursor >= 0 && m.cursor < len(m.results) {
		return m.results[m.cursor], true
	}
	return search.Result{}, false
}

// openLibrary entra en el modo biblioteca y refresca sus datos.
func (m *Model) openLibrary() {
	m.mode = modeLibrary
	m.libSection = sectionPlaylists
	m.libCursor = 0
	m.refreshLibrary()
	m.status = "Biblioteca: ←/→ sección · ↑/↓ navegar · esc volver."
}

// refreshLibrary recarga playlists, favoritos e historial desde los servicios.
func (m *Model) refreshLibrary() {
	if pls, err := m.playlists.List(); err != nil {
		m.warn("no se pudieron cargar playlists: " + err.Error())
		m.libPlaylists = nil
	} else {
		m.libPlaylists = pls
	}
	if favs, err := m.favorites.List(); err != nil {
		m.warn("no se pudieron cargar favoritos: " + err.Error())
		m.libFavorites = nil
	} else {
		m.libFavorites = favs
	}
	m.libHistory = nil
	for _, e := range m.history.Browse() {
		m.libHistory = append(m.libHistory, search.Result{ID: e.ID, Title: e.Title, Uploader: e.Uploader})
	}
}

// libCurrentLen devuelve el número de elementos de la sección activa.
func (m Model) libCurrentLen() int {
	switch m.libSection {
	case sectionPlaylists:
		return len(m.libPlaylists)
	case sectionFavorites:
		return len(m.libFavorites)
	case sectionHistory:
		return len(m.libHistory)
	}
	return 0
}

// updateLibraryMode gestiona la navegación y acciones del modo biblioteca.
func (m Model) updateLibraryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Library):
		m.mode = modeNormal
		m.status = "Pulsa / para buscar · L biblioteca."
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		_ = m.player.Close()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Down):
		if m.libCursor < m.libCurrentLen()-1 {
			m.libCursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.libCursor > 0 {
			m.libCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Next): // n: siguiente sección
		m.libSection = (m.libSection + 1) % librarySectionCount
		m.libCursor = 0
		return m, nil

	case key.Matches(msg, m.keys.Prev): // p: sección anterior
		m.libSection = (m.libSection + librarySectionCount - 1) % librarySectionCount
		m.libCursor = 0
		return m, nil

	case key.Matches(msg, m.keys.Favorite):
		if track, ok := m.libSelectedTrack(); ok {
			m.toggleFavorite(track)
			m.refreshLibrary()
			if m.libCursor >= m.libCurrentLen() && m.libCursor > 0 {
				m.libCursor = m.libCurrentLen() - 1
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.AddToPlaylist):
		if track, ok := m.libSelectedTrack(); ok {
			return m.openPlaylistPicker(track)
		}
		return m, nil

	case key.Matches(msg, m.keys.CreatePlaylist): // c abre el prompt de crear playlist
		m.mode = modeCreatePlaylist
		m.input.SetValue("")
		m.input.Placeholder = "Nombre de la playlist…"
		m.input.Prompt = "📝 "
		m.input.Focus()
		m.status = "Nombre de la nueva playlist · enter crear · esc cancelar."
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Enqueue): // enter reproduce la selección
		return m.libPlaySelection()
	}
	return m, nil
}

// updateCreatePlaylistMode gestiona el prompt de texto para crear una playlist.
func (m Model) updateCreatePlaylistMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel): // esc cancela y vuelve a biblioteca
		m.closeCreatePrompt()
		m.status = "Creación cancelada."
		return m, nil

	case key.Matches(msg, m.keys.Enqueue): // enter confirma el nombre
		name := m.input.Value()
		m.closeCreatePrompt()
		if _, err := m.playlists.Create(name); err != nil {
			m.status = m.styles.errorMsg.Render("No se pudo crear: " + err.Error())
			return m, nil
		}
		m.refreshLibrary()
		m.status = "Playlist creada: " + name
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// closeCreatePrompt cierra el prompt de creación y restaura la búsqueda en el
// input compartido, devolviendo la UI al modo biblioteca.
func (m *Model) closeCreatePrompt() {
	m.input.Blur()
	m.input.SetValue("")
	m.input.Placeholder = "Buscar canción…"
	m.input.Prompt = "🔎 "
	m.mode = modeLibrary
}

// libSelectedTrack devuelve la pista bajo el cursor en secciones de pistas
// (favoritos/historial). Para playlists no aplica (devuelve false).
func (m Model) libSelectedTrack() (search.Result, bool) {
	switch m.libSection {
	case sectionFavorites:
		if m.libCursor >= 0 && m.libCursor < len(m.libFavorites) {
			return m.libFavorites[m.libCursor], true
		}
	case sectionHistory:
		if m.libCursor >= 0 && m.libCursor < len(m.libHistory) {
			return m.libHistory[m.libCursor], true
		}
	}
	return search.Result{}, false
}

// libPlaySelection reproduce la selección: una playlist carga sus pistas en la
// cola; una pista de favoritos/historial se encola y reproduce.
func (m Model) libPlaySelection() (tea.Model, tea.Cmd) {
	switch m.libSection {
	case sectionPlaylists:
		if m.libCursor < 0 || m.libCursor >= len(m.libPlaylists) {
			return m, nil
		}
		pl := m.libPlaylists[m.libCursor]
		tracks, err := m.playlists.Play(pl.ID)
		if err != nil {
			m.status = m.styles.errorMsg.Render(err.Error())
			return m, nil
		}
		return m.enqueueAndPlay(tracks, "▶ playlist: "+pl.Name)

	case sectionFavorites, sectionHistory:
		if track, ok := m.libSelectedTrack(); ok {
			return m.enqueueAndPlay([]search.Result{track}, "▶ "+track.Title)
		}
	}
	return m, nil
}

// enqueueAndPlay añade pistas a la cola y arranca la reproducción si aún no había
// pista activa. Sale del modo biblioteca para mostrar la reproducción.
func (m Model) enqueueAndPlay(tracks []search.Result, status string) (tea.Model, tea.Cmd) {
	if len(tracks) == 0 {
		return m, nil
	}
	_, wasPlaying := m.queue.Current()
	for _, t := range tracks {
		m.queue.Add(t)
	}
	m.mode = modeNormal
	m.status = status
	if !wasPlaying {
		m.started = true
		if cur, ok := m.queue.Current(); ok {
			return m, loadTrackCmd(m.player, m.cache, cur)
		}
	}
	return m, nil
}

// openPlaylistPicker abre el picker de playlists (bubbles/list) para añadir la
// pista dada. Si no hay playlists, informa y no abre el picker.
func (m Model) openPlaylistPicker(track search.Result) (tea.Model, tea.Cmd) {
	pls, err := m.playlists.List()
	if err != nil {
		m.status = m.styles.errorMsg.Render(err.Error())
		return m, nil
	}
	if len(pls) == 0 {
		m.status = "No hay playlists. Crea una primero."
		return m, nil
	}
	items := make([]list.Item, 0, len(pls))
	for _, p := range pls {
		items = append(items, playlistItem{pl: p})
	}
	m.picker.SetItems(items)
	m.picker.Select(0)
	m.pickerTrack = track
	m.pickerReturn = m.mode
	m.mode = modePicker
	return m, nil
}

// updatePickerMode gestiona la selección de playlist para añadir una pista.
func (m Model) updatePickerMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = m.pickerReturn
		return m, nil

	case key.Matches(msg, m.keys.Enqueue):
		if it, ok := m.picker.SelectedItem().(playlistItem); ok {
			if err := m.playlists.Add(it.pl.ID, m.pickerTrack); err != nil {
				m.status = m.styles.errorMsg.Render(err.Error())
			} else {
				m.status = "Añadido a " + it.pl.Name + ": " + m.pickerTrack.Title
			}
		}
		m.mode = m.pickerReturn
		if m.mode == modeLibrary {
			m.refreshLibrary()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

// toggleFavorite alterna el favorito de track y actualiza el estado de la UI.
func (m *Model) toggleFavorite(track search.Result) {
	added, err := m.favorites.Toggle(track)
	if err != nil {
		m.status = m.styles.errorMsg.Render(err.Error())
		return
	}
	if added {
		m.status = "♥ favorito: " + track.Title
	} else {
		m.status = "Quitado de favoritos: " + track.Title
	}
}

// warn registra una advertencia en el logger si está disponible.
func (m *Model) warn(msg string) {
	if m.logger != nil {
		m.logger.Warn(msg)
	}
}

func (m Model) handlePlayerEvent(msg playerEventMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{waitForEventCmd(m.player)} // seguir escuchando
	switch msg.event.Kind {
	case player.EventEndFile:
		if m.queue.Next() {
			if cur, ok := m.queue.Current(); ok {
				cmds = append(cmds, loadTrackCmd(m.player, m.cache, cur))
			}
		} else {
			m.status = "Cola finalizada."
			// Al detenerse la reproducción se limpia la presencia de Discord para
			// no dejar una actividad "escuchando" obsoleta hasta salir de la app.
			if m.presence != nil {
				m.presence.Clear()
			}
		}
	case player.EventTrackChange:
		cmds = append(cmds, m.onTrackChange(msg.event.Track)...)
	}
	return m, tea.Batch(cmds...)
}

// onTrackChange reinicia el estado de los paneles de enriquecimiento para la
// nueva pista y abanica las Cmds de letra/portada/presencia/descarga-de-caché.
// Las Cmds de servicios apagados (nil) se filtran, de modo que con todos los
// toggles apagados no se dispara trabajo adicional (paridad con la Fase 2).
func (m *Model) onTrackChange(track search.Result) []tea.Cmd {
	m.curTrackID = track.ID
	m.curLyrics = lyrics.Lyrics{}
	m.curArtwork = ""
	m.lyricLine = -1

	var cmds []tea.Cmd
	if c := fetchLyricsCmd(m.lyrics, track); c != nil {
		cmds = append(cmds, c)
	}
	if c := renderArtworkCmd(m.artwork, track, m.artworkWidth(), m.artworkHeight()); c != nil {
		cmds = append(cmds, c)
	}
	if c := setPresenceCmd(m.presence, track); c != nil {
		cmds = append(cmds, c)
	}
	// Descargar a caché solo si la pista aún no está cacheada.
	if m.cache != nil {
		if _, ok := m.cache.Lookup(track.ID); ok {
			m.cachedIDs[track.ID] = true
		} else if c := cacheDownloadCmd(m.cache, track); c != nil {
			cmds = append(cmds, c)
		}
	}
	return cmds
}

// artworkWidth/artworkHeight dan dimensiones razonables para el panel de portada.
func (m Model) artworkWidth() int  { return 24 }
func (m Model) artworkHeight() int { return 12 }
