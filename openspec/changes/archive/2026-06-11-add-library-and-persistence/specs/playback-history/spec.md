# Delta for playback-history

## MODIFIED Requirements

### Requirement: Persist to Local Database

History MUST persist via the SQLite library storage layer
(`~/.local/share/terminaltube/library.db`), surviving restarts.
(Previously: history persisted to a JSON file `history.json`.)

#### Scenario: Survive restart

- GIVEN existe historial previo en la base de datos
- WHEN la app arranca
- THEN el historial se carga desde el almacenamiento SQLite

#### Scenario: Missing data

- GIVEN no existe historial almacenado
- WHEN la app arranca
- THEN se trata como historial vacío sin error

## ADDED Requirements

### Requirement: Migrate Legacy JSON History

On first run with the SQLite store, the system MUST import an existing
`history.json` into the database and MUST preserve the original file as a backup
(not delete it).

#### Scenario: Import existing JSON on first run

- GIVEN existe `history.json` y la base de datos no tiene historial
- WHEN la app arranca por primera vez con SQLite
- THEN las entradas se importan a la base de datos y `history.json` se conserva

#### Scenario: No legacy file

- GIVEN no existe `history.json`
- WHEN la app arranca
- THEN no se importa nada y el historial inicia vacío sin error

### Requirement: Browse History

The system MUST provide a navigable, time-ordered view of history entries (most recent
first) so the user can review previously played tracks.

#### Scenario: View ordered history

- GIVEN existen varias entradas de historial
- WHEN el usuario abre la vista de historial
- THEN se listan las pistas ordenadas de más reciente a más antigua

#### Scenario: Empty history view

- GIVEN no hay entradas de historial
- WHEN el usuario abre la vista de historial
- THEN se muestra una vista vacía sin error
