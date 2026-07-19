package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexcasdev/terminaltube/internal/lyrics"
	"github.com/alexcasdev/terminaltube/internal/metadata"
	"github.com/alexcasdev/terminaltube/internal/mpris"
	"github.com/alexcasdev/terminaltube/internal/player"
	"github.com/alexcasdev/terminaltube/internal/search"
)

// Update procesa mensajes y entrada de teclado.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.picker.SetSize(msg.Width, msg.Height-4)
		m.resultsList.SetSize(msg.Width, msg.Height-4)
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
		case modeURLInput:
			return m.updateURLInputMode(msg)
		case modeImportURL:
			return m.updateImportURLMode(msg)
		case modeImportName:
			return m.updateImportNameMode(msg)
		case modeLyricsSearch:
			return m.updateLyricsSearchMode(msg)
		case modeLyricsPicker:
			return m.updateLyricsPickerMode(msg)
		case modeResults:
			return m.updateResultsMode(msg)
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
			return m, nil
		}
		m.status = fmt.Sprintf("%d resultados.", len(m.results))
		// Guardia asíncrona: abrir el modal solo si la UI está en modo normal o
		// búsqueda. Un resultado que llega mientras el usuario navega la biblioteca
		// o tiene otro modal abierto no debe secuestrar el modo activo.
		if m.mode == modeNormal || m.mode == modeSearch {
			items := make([]list.Item, 0, len(m.results))
			for _, r := range m.results {
				items = append(items, resultItem{r: r, mark: m.cacheMark(r.ID)})
			}
			m.resultsList.SetItems(items)
			m.resultsList.Select(0)
			m.mode = modeResults
		}
		return m, nil

	case urlResolvedMsg:
		m.searching = false
		if msg.err != nil {
			m.status = m.styles.errorMsg.Render("No se pudo resolver la URL: " + msg.err.Error())
			return m, nil
		}
		// La pista resuelta se encola y se muestra como único resultado, de modo
		// que la tecla "a" pueda añadirla a una playlist existente reusando el picker.
		m.queue.Add(msg.track)
		m.results = []search.Result{msg.track}
		m.cursor = 0
		if !m.started {
			m.started = true
			m.status = "Reproduciendo desde URL: " + msg.track.Title + " · a → añadir a playlist"
			return m, loadTrackCmd(m.player, m.cache, msg.track)
		}
		m.status = "Añadido a la cola: " + msg.track.Title + " · a → añadir a playlist"
		return m, nil

	case playlistResolvedMsg:
		m.searching = false
		if msg.err != nil {
			m.status = m.styles.errorMsg.Render("No se pudo importar la playlist: " + msg.err.Error())
			return m, nil
		}
		if len(msg.tracks) == 0 {
			m.status = "La playlist no devolvió pistas."
			return m, nil
		}
		// Pistas resueltas: pedir al usuario el nombre de la playlist local.
		m.importTracks = msg.tracks
		m.importTitle = msg.title
		m.mode = modeImportName
		m.input.SetValue("")
		m.input.Placeholder = "Nombre de la playlist…"
		m.input.Prompt = "📝 "
		m.input.Focus()
		info := fmt.Sprintf("%d pistas", len(msg.tracks))
		if msg.title != "" {
			info += " · " + msg.title
		}
		m.status = "Importar (" + info + "): teclea un nombre · enter crear · esc cancelar."
		return m, textinput.Blink

	case lyricsCandidatesMsg:
		m.searching = false
		if msg.err != nil {
			m.status = m.styles.errorMsg.Render("Error buscando letra: " + msg.err.Error())
			return m, nil
		}
		if len(msg.cands) == 0 {
			m.status = "Sin resultados de letra."
			return m, nil
		}
		m.lyricCands = msg.cands
		items := make([]list.Item, 0, len(msg.cands))
		for _, c := range msg.cands {
			items = append(items, candidateItem{c: c})
		}
		m.picker.SetItems(items)
		m.picker.Title = "Elige la letra"
		m.picker.Select(0)
		m.mode = modeLyricsPicker
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

	case mpris.PlayPauseMsg:
		m.togglePause()
		return m, nil

	case mpris.NextMsg:
		if m.queue.Next() {
			if cur, ok := m.queue.Current(); ok {
				return m, loadTrackCmd(m.player, m.cache, cur)
			}
		}
		return m, nil

	case mpris.PrevMsg:
		if m.queue.Prev() {
			if cur, ok := m.queue.Current(); ok {
				return m, loadTrackCmd(m.player, m.cache, cur)
			}
		}
		return m, nil

	case mpris.StopMsg:
		if err := m.player.Stop(); err != nil {
			m.status = m.styles.errorMsg.Render(err.Error())
		}
		if m.mpris != nil {
			m.mpris.SetPlaybackStatus("Stopped")
		}
		return m, nil

	case mpris.SeekMsg:
		offsetSec := float64(msg.Offset) / 1e6
		if err := m.player.Seek(offsetSec); err != nil {
			m.status = m.styles.errorMsg.Render(err.Error())
		}
		if m.mpris != nil {
			newPos := m.pos + offsetSec
			if newPos < 0 {
				newPos = 0
			}
			if newPos > m.dur {
				newPos = m.dur
			}
			m.mpris.Seeked(int64(newPos * 1e6))
		}
		return m, nil

	case mpris.SetVolumeMsg:
		target := int(msg.Volume * 130)
		v, err := m.player.AddVolume(target - m.player.Volume())
		if err != nil {
			m.status = m.styles.errorMsg.Render(err.Error())
			return m, nil
		}
		if m.mpris != nil {
			m.mpris.SetVolume(v)
		}
		return m, nil

	case tickMsg:
		// El sondeo de posición corre en su propio Cmd para no bloquear Update.
		return m, tea.Batch(fetchPositionCmd(m.player), tickCmd())

	case animTickMsg:
		// Avanza la animación del visualizador solo mientras hay reproducción;
		// en pausa o sin pista el frame se congela y las barras quedan planas.
		if m.isPlaying() {
			m.animFrame++
		}
		return m, animTickCmd()

	case posMsg:
		m.pos, m.dur = msg.pos, msg.dur
		m.advanceLyric()
		if m.mpris != nil {
			m.mpris.SetPosition(m.pos)
		}
		return m, nil

	case lyricsMsg:
		// Descartar respuestas obsoletas de una pista ya cambiada.
		if msg.videoID == m.curTrackID {
			m.curLyrics = msg.lyrics
			m.lyricLine = -1
			m.advanceLyric()
			if m.mpris != nil {
				if cur, ok := m.queue.Current(); ok {
					m.mpris.SetMetadata(cur, m.curLyrics)
				}
			}
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

// togglePause alterna pausa/reproducción y actualiza el estado MPRIS.
func (m *Model) togglePause() {
	if _, ok := m.queue.Current(); !ok {
		return // nada en reproducción que pausar
	}
	if err := m.player.TogglePause(); err != nil {
		m.status = m.styles.errorMsg.Render(err.Error())
		return
	}
	if m.mpris != nil {
		if m.player.Paused() {
			m.mpris.SetPlaybackStatus("Paused")
		} else {
			m.mpris.SetPlaybackStatus("Playing")
		}
	}
}

// isPlaying indica si hay una pista activa que no está en pausa. El
// visualizador de barras solo se anima en ese estado.
func (m Model) isPlaying() bool {
	if _, ok := m.queue.Current(); !ok {
		return false
	}
	return !m.player.Paused()
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
		m.togglePause()
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
		if m.mpris != nil {
			m.mpris.SetVolume(v)
		}
		return m, nil

	case key.Matches(msg, m.keys.VolDown):
		v, _ := m.player.AddVolume(-5)
		m.status = fmt.Sprintf("Volumen: %d", v)
		if m.mpris != nil {
			m.mpris.SetVolume(v)
		}
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

	case key.Matches(msg, m.keys.AddFromURL):
		if m.resolver == nil {
			m.status = "Resolución por URL no disponible."
			return m, nil
		}
		m.mode = modeURLInput
		m.input.SetValue("")
		m.input.Placeholder = "URL de vídeo de YouTube…"
		m.input.Prompt = "🔗 "
		m.input.Focus()
		m.status = "Pega una URL de vídeo · enter añadir · esc cancelar."
		return m, textinput.Blink

	case key.Matches(msg, m.keys.ImportPlaylist):
		if m.plResolver == nil {
			m.status = "Importar playlist no disponible."
			return m, nil
		}
		m.mode = modeImportURL
		m.input.SetValue("")
		m.input.Placeholder = "URL de playlist de YouTube…"
		m.input.Prompt = "🔗 "
		m.input.Focus()
		m.status = "Pega una URL de playlist · enter importar · esc cancelar."
		return m, textinput.Blink

	case key.Matches(msg, m.keys.LyricsSearch):
		cur, ok := m.queue.Current()
		if !ok {
			m.status = "No hay pista en reproducción para buscar su letra."
			return m, nil
		}
		if m.lyrics == nil {
			m.status = "Letras desactivadas."
			return m, nil
		}
		m.lyricsTrack = cur
		m.mode = modeLyricsSearch
		// Prellenar con el título/artista normalizados de la pista actual.
		artist, title := metadata.Normalize(cur)
		q := strings.TrimSpace(artist + " " + title)
		m.input.SetValue(q)
		m.input.CursorEnd()
		m.input.Placeholder = "Buscar letra…"
		m.input.Prompt = "🔍 "
		m.input.Focus()
		m.status = "Ajusta la consulta de letra · enter buscar · esc cancelar."
		return m, textinput.Blink

	case key.Matches(msg, m.keys.ClearQueue):
		return m.clearQueue()
	}
	return m, nil
}

// clearQueue vacía la cola completa, detiene la reproducción y reinicia el estado
// de "ahora suena" (letra/portada/presencia).
func (m Model) clearQueue() (tea.Model, tea.Cmd) {
	if m.queue.Len() == 0 {
		m.status = "La cola ya está vacía."
		return m, nil
	}
	m.queue.Clear()
	if err := m.player.Stop(); err != nil {
		m.warn("no se pudo detener la reproducción: " + err.Error())
	}
	if m.mpris != nil {
		m.mpris.SetPlaybackStatus("Stopped")
	}
	m.started = false
	m.curTrackID = ""
	m.curLyrics = lyrics.Lyrics{}
	m.curArtwork = ""
	m.lyricLine = -1
	m.pos, m.dur = 0, 0
	if m.presence != nil {
		m.presence.Clear()
	}
	m.status = "Cola limpiada."
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

// endInput cierra un prompt de texto y restaura los valores de búsqueda en el
// input compartido, devolviendo la UI al modo normal.
func (m *Model) endInput() {
	m.input.Blur()
	m.input.SetValue("")
	m.input.Placeholder = "Buscar canción…"
	m.input.Prompt = "🔎 "
	m.mode = modeNormal
}

// restorePickerTitle restaura el título por defecto del picker (compartido entre
// la selección de playlist y la de candidatos de letra).
func (m *Model) restorePickerTitle() { m.picker.Title = "Añadir a playlist" }

// updateURLInputMode gestiona el prompt de URL de vídeo: al enviar, resuelve la
// URL en segundo plano.
func (m Model) updateURLInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.endInput()
		m.status = "Cancelado."
		return m, nil
	case key.Matches(msg, m.keys.Enqueue):
		raw := strings.TrimSpace(m.input.Value())
		m.endInput()
		if raw == "" {
			return m, nil
		}
		if m.resolver == nil {
			m.status = "Resolución por URL no disponible."
			return m, nil
		}
		m.searching = true
		m.status = "Resolviendo URL…"
		return m, resolveURLCmd(m.resolver, raw)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// updateImportURLMode gestiona el prompt de URL de playlist: al enviar, resuelve
// la playlist en segundo plano (luego se pedirá el nombre).
func (m Model) updateImportURLMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.endInput()
		m.status = "Cancelado."
		return m, nil
	case key.Matches(msg, m.keys.Enqueue):
		raw := strings.TrimSpace(m.input.Value())
		m.endInput()
		if raw == "" {
			return m, nil
		}
		if m.plResolver == nil {
			m.status = "Importar playlist no disponible."
			return m, nil
		}
		m.searching = true
		m.status = "Resolviendo playlist…"
		return m, resolvePlaylistCmd(m.plResolver, raw)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// updateImportNameMode gestiona el prompt de nombre tras resolver una playlist:
// crea la playlist local y le añade las pistas resueltas. Ante un nombre inválido
// o duplicado se mantiene en el prompt para que el usuario corrija.
func (m Model) updateImportNameMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.importTracks = nil
		m.importTitle = ""
		m.endInput()
		m.status = "Importación cancelada."
		return m, nil
	case key.Matches(msg, m.keys.Enqueue):
		name := strings.TrimSpace(m.input.Value())
		id, err := m.playlists.Create(name)
		if err != nil {
			// Nombre vacío o duplicado: permanecer en el prompt para corregir.
			m.status = m.styles.errorMsg.Render("No se pudo crear: " + err.Error())
			return m, nil
		}
		tracks := m.importTracks
		m.importTracks = nil
		m.importTitle = ""
		m.endInput()
		added := 0
		for _, t := range tracks {
			if err := m.playlists.Add(id, t); err != nil {
				m.warn("no se pudo añadir pista a la playlist importada: " + err.Error())
				continue
			}
			added++
		}
		m.status = fmt.Sprintf("Playlist importada: %s (%d pistas)", name, added)
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// updateLyricsSearchMode gestiona el prompt de consulta de la búsqueda manual de
// letra.
func (m Model) updateLyricsSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.endInput()
		m.status = "Búsqueda de letra cancelada."
		return m, nil
	case key.Matches(msg, m.keys.Enqueue):
		q := strings.TrimSpace(m.input.Value())
		m.endInput()
		if q == "" {
			return m, nil
		}
		if m.lyrics == nil {
			m.status = "Letras desactivadas."
			return m, nil
		}
		m.searching = true
		m.status = "Buscando letra…"
		return m, searchLyricsCmd(m.lyrics, q)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// updateLyricsPickerMode gestiona la selección de un candidato de letra. Al
// elegir, fija la letra y persiste la referencia para la pista.
func (m Model) updateLyricsPickerMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.restorePickerTitle()
		m.mode = modeNormal
		m.status = "Búsqueda de letra cancelada."
		return m, nil
	case key.Matches(msg, m.keys.Enqueue):
		it, ok := m.picker.SelectedItem().(candidateItem)
		m.restorePickerTitle()
		m.mode = modeNormal
		if !ok {
			return m, nil
		}
		m.status = "Letra fijada para: " + m.lyricsTrack.Title
		return m, selectLyricsCmd(m.lyrics, m.lyricsTrack, it.c)
	}
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
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

// updateResultsMode gestiona la navegación y acciones del modal de resultados de
// búsqueda (modeResults). Espeja la estructura de updatePickerMode.
func (m Model) updateResultsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyCtrlC:
		// ctrl+c hace hard-quit en cualquier modo; cerramos limpio el reproductor.
		m.quitting = true
		_ = m.player.Close()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Cancel):
		// Esc cierra el modal y vuelve al modo normal sin encolar nada.
		m.mode = modeNormal
		return m, nil

	case key.Matches(msg, m.keys.Enqueue):
		// Enter encola el resultado seleccionado y cierra el modal.
		it, ok := m.resultsList.SelectedItem().(resultItem)
		m.mode = modeNormal
		if !ok {
			return m, nil
		}
		track := it.r
		m.queue.Add(track)
		if !m.started {
			m.started = true
			return m, loadTrackCmd(m.player, m.cache, track)
		}
		m.status = "Añadido a la cola: " + track.Title
		return m, nil

	case key.Matches(msg, m.keys.AddToPlaylist):
		// `a` abre el picker de playlists; al volver debe retornar al modal.
		it, ok := m.resultsList.SelectedItem().(resultItem)
		if !ok {
			return m, nil
		}
		return m.openPlaylistPicker(it.r)

	case key.Matches(msg, m.keys.Favorite):
		// `f` alterna favorito sin cerrar el modal.
		if it, ok := m.resultsList.SelectedItem().(resultItem); ok {
			m.toggleFavorite(it.r)
		}
		return m, nil
	}

	// Navegación (↑/↓/j/k) y cualquier tecla de lista se delegan al componente.
	var cmd tea.Cmd
	m.resultsList, cmd = m.resultsList.Update(msg)
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
			if m.mpris != nil {
				m.mpris.SetPlaybackStatus("Stopped")
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

	if m.mpris != nil {
		m.mpris.SetMetadata(track, lyrics.Lyrics{})
		m.mpris.SetPlaybackStatus("Playing")
	}

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
