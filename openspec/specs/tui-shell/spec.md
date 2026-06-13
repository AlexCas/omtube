# TUI Shell Specification

## Purpose

Interfaz de terminal (Bubble Tea + Lip Gloss) que integra búsqueda, resultados, cola
y controles por teclado en una sola pantalla.

## Requirements

### Requirement: Search Input Mode

The system MUST provide a search mode toggled with `/` where typed text is captured
and submitting runs a search and shows results.

#### Scenario: Enter and submit search

- GIVEN la TUI está en modo normal
- WHEN el usuario pulsa `/`, escribe una consulta y la envía
- THEN se ejecuta la búsqueda y los resultados se muestran en su panel

### Requirement: Search Results Modal

The system MUST present search results in a full-screen `modeResults` takeover
modal that is entered ONLY after a multi-result search completes. The modal MUST
own the entire viewport while active and MUST be the sole surface where results
are browsed and acted upon. Results-mode key hints MUST be shown ONLY while the
modal is active (D4).

#### Scenario: Open modal on multi-result search

- GIVEN the TUI is in normal or search mode
- WHEN a search returns multiple results
- THEN the `modeResults` modal opens full-screen with the results
- AND the main view is hidden until the modal is closed

#### Scenario: Dismiss with Esc

- GIVEN the results modal is active
- WHEN the user presses Esc
- THEN the modal closes and returns to the main view without enqueuing anything

#### Scenario: Enqueue with Enter

- GIVEN the results modal is active with a selection
- WHEN the user presses Enter
- THEN the selected track is added to the queue and the modal returns to the main view

#### Scenario: Navigate results

- GIVEN the results modal is active
- WHEN the user presses Up/Down or `j`/`k`
- THEN the selection moves through the results list

#### Scenario: Add selection to a playlist

- GIVEN the results modal is active with a selection
- WHEN the user presses `a` and chooses a playlist
- THEN the track is added to that playlist and the flow returns to the results modal (not the library)

#### Scenario: Toggle favorite on selection

- GIVEN the results modal is active with a selection
- WHEN the user presses `f`
- THEN the favorite status of the selected track is toggled and the modal remains active

#### Scenario: Results hints visible only in modal

- GIVEN the TUI is in the main view (no modal active)
- WHEN the help line is displayed
- THEN the results-mode key hints are not shown
- AND those hints only appear while `modeResults` is active

### Requirement: Results and Queue Panels

The system MUST display search results in a full-screen `modeResults` takeover
modal entered only after a multi-result search, and MUST display the playback
queue as an inline full-size panel always visible in the main view, highlighting
the current track. The results panel MUST NOT be drawn in the main view. Entering
search mode (`/`) MUST present a fresh search input and MUST NOT reopen the previous
results; submitting a non-empty query MUST run a new search and rebuild the results
modal from scratch, while submitting an empty query MUST start no search. There is
no persistent "reopen last results" affordance (D2). When the lyrics and/or
artwork enrichment services are active, their panels MUST be drawn in the same
horizontal row as the queue (queue | lyrics | cover) rather than stacked below
it; when both enrichment services are off, the row MUST contain only the queue.
(Previously: results and the queue were drawn side-by-side as always-visible panels in the main view.)

#### Scenario: Enqueue from results

- GIVEN results are visible in the `modeResults` modal
- WHEN the user selects a result and confirms
- THEN the track is added to the queue (and played if the queue was empty)

#### Scenario: Queue always visible inline

- GIVEN the TUI is in the main view
- WHEN no modal is active
- THEN the queue is shown as a full-size inline panel
- AND the results panel is not drawn in the main view

#### Scenario: Enrichment panels beside the queue

- GIVEN the TUI is in the main view with the lyrics and/or artwork services active
- WHEN the main view is rendered
- THEN the lyrics and cover panels are drawn in the same horizontal row as the queue (queue | lyrics | cover)
- AND when both enrichment services are off the row contains only the queue

#### Scenario: Now playing centered under enrichment

- GIVEN the TUI is in the main view with the enrichment panels active
- WHEN the now-playing line (current track title, progress and volume) is rendered
- THEN it is centered under the lyrics+cover block (indented to the right of the queue)
- AND when both enrichment services are off it is left-aligned under the queue as before

#### Scenario: Opening search does not reopen previous results

- GIVEN results from a previous search existed
- WHEN the user presses `/` to enter search mode
- THEN a fresh search input is shown and the previous results are not reopened
- AND submitting an empty query starts no search
- AND submitting a non-empty query runs a new search and rebuilds the results modal

### Requirement: Keyboard Controls

The system MUST bind: espacio (play/pausa), `n` (siguiente), `p` (anterior),
`+`/`-` (volumen), `/` (buscar), `q` (salir).

#### Scenario: Quit cleanly

- GIVEN la app está corriendo
- WHEN el usuario pulsa `q`
- THEN mpv se detiene, el socket se cierra y la app termina sin errores

#### Scenario: Toggle playback

- GIVEN una pista en reproducción
- WHEN el usuario pulsa espacio
- THEN la reproducción alterna entre pausa y play

### Requirement: Non-blocking UI

The system MUST run search and playback resolution without freezing the UI, using
asynchronous commands.

#### Scenario: Search does not block

- GIVEN una búsqueda en curso
- WHEN yt-dlp tarda en responder
- THEN la UI sigue respondiendo y muestra estado de carga

### Requirement: Library Mode

The system MUST provide a library mode/panel that lets the user browse playlists,
favorites, and history, and switch back to the normal/search view.

#### Scenario: Open and close library mode

- GIVEN la TUI está en modo normal
- WHEN el usuario abre el modo biblioteca
- THEN se muestra la vista de biblioteca con playlists, favoritos e historial
- AND el usuario puede volver al modo normal

#### Scenario: Navigate library sections

- GIVEN la TUI está en modo biblioteca
- WHEN el usuario navega entre playlists, favoritos e historial
- THEN se muestra la sección seleccionada y su contenido

### Requirement: Create Playlist from UI

The system MUST let the user create a new playlist from the library mode by entering a
name. The shortcut MUST NOT collide with existing controls (espacio, `n`, `p`, `+`,
`-`, `/`, `q`).

#### Scenario: Create a playlist by name

- GIVEN la TUI está en modo biblioteca
- WHEN el usuario pulsa el atajo de crear playlist, escribe un nombre y confirma
- THEN se crea una playlist vacía con ese nombre y aparece en la lista de playlists

#### Scenario: Reject empty or duplicate name

- GIVEN la TUI está en el prompt de crear playlist
- WHEN el usuario confirma un nombre vacío o ya existente
- THEN no se crea la playlist y la UI informa del motivo

### Requirement: Library Action Shortcuts

The system MUST bind shortcuts to manage the library: marcar/desmarcar favorito de la
pista seleccionada y añadir la pista seleccionada a una playlist. Shortcuts MUST NOT
collide with existing controls (espacio, `n`, `p`, `+`, `-`, `/`, `q`).

#### Scenario: Toggle favorite from the UI

- GIVEN hay una pista seleccionada
- WHEN el usuario pulsa el atajo de favorito
- THEN el estado de favorito de la pista se alterna y la UI lo refleja

#### Scenario: Add selected track to a playlist

- GIVEN hay una pista seleccionada y existe al menos una playlist
- WHEN el usuario pulsa el atajo de añadir a playlist y elige una
- THEN la pista se añade a esa playlist y la UI confirma la acción

#### Scenario: Play a playlist from the UI

- GIVEN el usuario está viendo una playlist con pistas en el modo biblioteca
- WHEN el usuario la reproduce
- THEN sus pistas se cargan en la cola en orden y comienza la reproducción

### Requirement: Lyrics Panel

The system MUST provide a panel that displays the current track's lyrics and, when
synced lyrics are available, highlights the line matching playback position. The panel
MUST show a clear "sin letra" state when none are available, and MUST NOT block the UI.

#### Scenario: Show lyrics

- GIVEN hay letra disponible para la pista actual
- WHEN el panel de letra está visible
- THEN se muestra la letra; si es sincronizada, se resalta la línea actual

#### Scenario: No lyrics state

- GIVEN no hay letra disponible para la pista
- WHEN el panel de letra está visible
- THEN se muestra un estado "sin letra" sin afectar el resto de la UI

### Requirement: Artwork Panel

The system MUST provide a panel that displays the current track's artwork using the
terminal's image capability, degrading gracefully (chafa o sin-portada) when images are
unsupported.

#### Scenario: Show artwork

- GIVEN el terminal soporta imágenes y hay portada disponible
- WHEN el panel de portada está visible
- THEN se renderiza la portada de la pista actual

#### Scenario: Degrade without image support

- GIVEN el terminal no soporta imágenes
- WHEN el panel de portada está visible
- THEN se muestra una degradación (chafa/placeholder) sin error

### Requirement: Cache Indicator

The system MUST indicate, per track in results/queue, whether it is available in the
local cache.

#### Scenario: Show cached status

- GIVEN una pista está cacheada localmente
- WHEN se muestra en resultados o en la cola
- THEN la UI muestra un indicador de "cacheada" para esa pista

### Requirement: Add by URL Input Mode

The TUI MUST provide a mode to paste a YouTube video URL. On submit, it MUST resolve the
URL, append the resolved track to the queue, and present that track so the user can add
it to an existing playlist via the existing add-to-playlist picker. A single track
resolved from a URL MUST NOT open the `modeResults` modal; the TUI MUST stay in the main
view (D1). Resolution MUST be non-blocking and surface a readable error on failure.
(Previously: did not constrain modal behavior for a single URL-resolved track.)

#### Scenario: Paste a video URL

- GIVEN the user opens the "add by URL" mode
- WHEN they paste a video URL and submit it
- THEN the resolved track is enqueued and displayed as a selectable result
- AND the TUI remains in the main view without opening the results modal

#### Scenario: Add the URL track to a playlist

- GIVEN a track freshly resolved from a URL is displayed
- WHEN the user presses the add-to-playlist action
- THEN the existing playlist picker opens for that track

#### Scenario: Invalid URL feedback

- GIVEN the user submits a URL that cannot be resolved
- WHEN resolution fails
- THEN a readable error is displayed and the TUI remains operational

### Requirement: Import Playlist Mode

The TUI MUST provide a mode to paste a YouTube playlist URL. After resolving the
playlist, it MUST prompt the user to type a name for the new local playlist, then create
it with the resolved tracks. Empty names and name collisions MUST be reported without
creating a playlist.

#### Scenario: Import a playlist

- GIVEN el usuario abre el modo "importar playlist" y pega una URL de playlist
- WHEN se resuelve y teclea un nombre nuevo
- THEN se crea la playlist local con sus pistas y se confirma en la barra de estado

#### Scenario: Import name rejected

- GIVEN una playlist resuelta a la espera de nombre
- WHEN el usuario teclea un nombre vacío o ya existente
- THEN no se crea la playlist y se informa el error

### Requirement: Manual Lyrics Search Mode

The TUI MUST provide a mode to manually search lyrics for the current track: enter a
query, view candidates, and select one. Selecting a candidate MUST update the lyrics
panel and persist the chosen reference for the track.

#### Scenario: Open manual lyrics search

- GIVEN hay una pista en reproducción
- WHEN el usuario abre la búsqueda manual de letra
- THEN puede teclear una consulta prellenada con el título/artista actual

#### Scenario: Pick a lyrics candidate

- GIVEN se muestran candidatos de letra
- WHEN el usuario elige uno
- THEN el panel de letra se actualiza y la referencia queda guardada para la pista

### Requirement: Clear Queue Shortcut

The TUI MUST provide a keyboard shortcut to clear the queue, which empties the queue,
stops playback, and resets the now-playing panels (lyrics/artwork).

#### Scenario: Clear the queue from the UI

- GIVEN hay pistas en la cola y una reproduciéndose
- WHEN el usuario pulsa la tecla de limpiar cola
- THEN la cola se vacía, la reproducción se detiene, los paneles se limpian y la barra de estado lo confirma
