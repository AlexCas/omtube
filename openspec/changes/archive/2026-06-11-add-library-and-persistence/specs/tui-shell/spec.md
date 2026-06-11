# Delta for tui-shell

## ADDED Requirements

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
