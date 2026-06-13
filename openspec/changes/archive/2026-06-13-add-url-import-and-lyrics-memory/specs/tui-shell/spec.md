# Delta for tui-shell

## ADDED Requirements

### Requirement: Add by URL Input Mode

The TUI MUST provide a mode to paste a YouTube video URL. On submit, it MUST resolve the
URL, append the resolved track to the queue, and present that track so the user can add
it to an existing playlist via the existing add-to-playlist picker. Resolution MUST be
non-blocking and surface a readable error on failure.

#### Scenario: Paste a video URL

- GIVEN el usuario abre el modo "añadir por URL"
- WHEN pega una URL de vídeo y la envía
- THEN la pista resuelta se encola y se muestra como resultado seleccionable

#### Scenario: Add the URL track to a playlist

- GIVEN una pista recién resuelta por URL está mostrada
- WHEN el usuario pulsa la acción de añadir a playlist
- THEN se abre el picker de playlists existentes para esa pista

#### Scenario: Invalid URL feedback

- GIVEN el usuario envía una URL no resoluble
- WHEN falla la resolución
- THEN se muestra un error legible y la TUI sigue operativa

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
