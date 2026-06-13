# Delta for playlists

## ADDED Requirements

### Requirement: Import Playlist from YouTube URL

The system MUST allow importing a YouTube playlist URL as a new local playlist. The
local playlist name MUST be provided by the user (not derived from the YouTube title)
and is subject to the same non-empty and uniqueness rules as creating a playlist. All
resolved tracks MUST be added to the new playlist in their resolved order; duplicate
tracks within the import MUST NOT create duplicate entries.

#### Scenario: Import into a new local playlist

- GIVEN una URL de playlist de YouTube con varias pistas
- WHEN el usuario la importa y teclea un nombre nuevo "Workout"
- THEN se crea la playlist "Workout" con todas las pistas en orden

#### Scenario: Import name collides with existing playlist

- GIVEN ya existe una playlist "Workout"
- WHEN el usuario importa y teclea "Workout"
- THEN no se crea ni se importan pistas y se informa el conflicto

#### Scenario: Import with empty name

- GIVEN el usuario importa una playlist
- WHEN el nombre tecleado está vacío o en blanco
- THEN no se crea y se informa el error

#### Scenario: Import resolves no tracks

- GIVEN una URL de playlist sin pistas accesibles
- WHEN el usuario la importa
- THEN no se crea una playlist vacía y se informa que no se obtuvieron pistas
