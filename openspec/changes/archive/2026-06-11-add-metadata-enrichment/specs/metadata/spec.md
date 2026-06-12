# Metadata Specification

## Purpose

Proveer un normalizador puro y determinista que derive un par `(artist, title)` limpio
a partir de un `(Title, Uploader)` crudo de YouTube, exclusivamente para las consultas
salientes de letra y portada. Nunca muta datos almacenados y es completamente testeable
por tablas, sin I/O.

## Requirements

### Requirement: Normalize Query Metadata

The system MUST derive a clean `(artist, title)` from a raw `search.Result`
`(Title, Uploader)` for use as query input only. The function MUST be pure and
deterministic (no network, no I/O) and MUST collapse repeated/leading/trailing
whitespace in its outputs.

#### Scenario: Split artist and title

- GIVEN un título de la forma `"Artist - Song"` (con separador `" - "` o `" – "`)
- WHEN se normaliza el título
- THEN el texto antes del primer separador se vuelve el artista
- AND el texto posterior se vuelve el título

#### Scenario: Strip suffix and feat noise

- GIVEN un título como `"Artist - Song (Official Music Video) feat. Other"`
- WHEN se normaliza el título
- THEN se eliminan del título las etiquetas entre paréntesis/corchetes (`(Official Video)`,
  `(Official Music Video)`, `[MV]`, `(Lyrics)`, `(Lyric Video)`, `(Audio)`, `(Visualizer)`,
  `(HD)`, etiquetas de año)
- AND se descartan los segmentos `feat.`/`ft.` de la consulta de letra/portada

#### Scenario: Derive artist from channel when no split

- GIVEN un título sin separador `" - "`/`" – "` y un uploader como `"ArtistVEVO"` o `"Artist - Topic"`
- WHEN se normalizan los metadatos
- THEN el artista se deriva del uploader quitando el ruido `VEVO`, `- Topic` y `Official`
- AND el título es el título completo ya limpio

### Requirement: Query-Only, Non-Mutating

The system MUST NOT mutate the source `search.Result` or any stored library data
(history, favorites, playlists, cache rows). Normalized values MUST be used solely as
outbound query input.

#### Scenario: Stored data unchanged

- GIVEN un `search.Result` crudo se normaliza para una consulta
- WHEN termina la normalización
- THEN los campos originales `Title` y `Uploader` quedan sin cambios
- AND las filas almacenadas de historial/favoritos/playlists siguen mostrando el título original de YouTube
