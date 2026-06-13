# Playlists Specification

## Purpose

Gestión de listas locales de pistas: crear, renombrar, borrar, añadir/quitar pistas y
reproducir una playlist como cola. Los datos persisten entre sesiones.

## Requirements

### Requirement: Create Playlist

The system MUST allow creating a named playlist. Playlist names MUST be non-empty;
duplicate names MUST be rejected.

#### Scenario: Create with a valid name

- GIVEN no existe una playlist llamada "Focus"
- WHEN el usuario crea una playlist "Focus"
- THEN la playlist queda registrada y aparece en la lista de playlists

#### Scenario: Reject empty name

- GIVEN el usuario intenta crear una playlist
- WHEN el nombre está vacío o en blanco
- THEN no se crea y se informa el error

#### Scenario: Reject duplicate name

- GIVEN ya existe una playlist "Focus"
- WHEN el usuario crea otra "Focus"
- THEN no se crea y se informa el conflicto

### Requirement: Rename Playlist

The system MUST allow renaming an existing playlist, subject to the same non-empty and
uniqueness rules as creation.

#### Scenario: Rename to a free name

- GIVEN existe una playlist "Focus"
- WHEN el usuario la renombra a "Deep Work"
- THEN la playlist conserva sus pistas bajo el nuevo nombre

#### Scenario: Rename collision

- GIVEN existen playlists "Focus" y "Chill"
- WHEN el usuario renombra "Focus" a "Chill"
- THEN no se renombra y se informa el conflicto

### Requirement: Delete Playlist

The system MUST allow deleting a playlist, removing it and its track membership without
affecting the underlying tracks or other playlists.

#### Scenario: Delete an existing playlist

- GIVEN existe una playlist "Focus" con pistas
- WHEN el usuario la borra
- THEN desaparece de la lista y sus membresías se eliminan

#### Scenario: Delete non-existent playlist

- GIVEN no existe una playlist con ese identificador
- WHEN el usuario intenta borrarla
- THEN no hay cambios y se informa que no existe

### Requirement: Manage Playlist Tracks

The system MUST allow adding and removing tracks from a playlist. Adding a track already
present MUST NOT create a duplicate entry. Track order MUST be preserved as added.

#### Scenario: Add a track

- GIVEN una playlist "Focus" sin esa pista
- WHEN el usuario añade una pista
- THEN la pista aparece al final de la playlist

#### Scenario: Add duplicate track

- GIVEN la pista ya está en "Focus"
- WHEN el usuario la añade de nuevo
- THEN la playlist no cambia (sin duplicado)

#### Scenario: Remove a track

- GIVEN la pista está en "Focus"
- WHEN el usuario la quita
- THEN la pista deja de pertenecer a la playlist y el orden del resto se conserva

### Requirement: Play Playlist as Queue

The system MUST load a playlist's tracks into the playback queue in stored order so the
playlist can be reproduced.

#### Scenario: Play a populated playlist

- GIVEN una playlist "Focus" con varias pistas
- WHEN el usuario la reproduce
- THEN sus pistas se cargan en la cola en orden y comienza la reproducción

#### Scenario: Play an empty playlist

- GIVEN una playlist sin pistas
- WHEN el usuario la reproduce
- THEN la cola no cambia y se informa que la playlist está vacía

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
