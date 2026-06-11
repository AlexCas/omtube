# Delta for playback-history

## ADDED Requirements

### Requirement: Record Played Tracks

The system MUST append a track to history when it starts playing (id, title,
uploader, ISO-8601 timestamp).

#### Scenario: Record on play

- GIVEN una pista comienza
- WHEN inicia la reproducción
- THEN se añade una entrada con timestamp

### Requirement: Persist to JSON File

History MUST persist to `~/.local/share/terminaltube/history.json` across restarts.

#### Scenario: Survive restart

- GIVEN historial previo en disco
- WHEN la app arranca
- THEN se carga desde el JSON

#### Scenario: Missing file

- GIVEN no existe el archivo
- WHEN la app arranca
- THEN historial vacío sin error
