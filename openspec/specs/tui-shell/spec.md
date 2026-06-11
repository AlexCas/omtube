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

### Requirement: Results and Queue Panels

The system MUST display search results and the playback queue, highlighting the
current track.

#### Scenario: Enqueue from results

- GIVEN hay resultados visibles
- WHEN el usuario selecciona un resultado
- THEN la pista se añade a la cola (y se reproduce si la cola estaba vacía)

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
