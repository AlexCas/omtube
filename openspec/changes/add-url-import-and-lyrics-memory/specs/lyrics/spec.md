# Delta for lyrics

## MODIFIED Requirements

### Requirement: Fetch Lyrics

The system MUST fetch lyrics for the current track from a community API. If a saved
lyrics reference (a remembered query and/or provider id) exists for the track, the
system MUST resolve lyrics using that saved reference instead of the auto-derived query.
Otherwise it uses a normalized `(artist, title)` query rather than the raw YouTube
title/uploader, preferring synced (.lrc) lyrics and falling back to plain text. On a
strict `/api/get` miss, the system MUST retry the same provider's fuzzy `/api/search`
endpoint with the same normalized query before declaring "no lyrics". Fetching MUST be
controllable by a config toggle, and results SHOULD be cached.
(Previously: always used the auto-derived normalized query; no saved reference was consulted.)

#### Scenario: Saved reference reused

- GIVEN existe una referencia de letra guardada para la pista
- WHEN suena la pista
- THEN la letra se resuelve con la consulta/referencia guardada, no con la consulta automática

#### Scenario: Synced lyrics found

- GIVEN el toggle de letras está activo y existe letra sincronizada
- WHEN suena una pista
- THEN se obtiene y parsea el .lrc con marcas de tiempo

#### Scenario: Plain lyrics fallback

- GIVEN no hay letra sincronizada pero sí texto plano
- WHEN suena la pista
- THEN se muestra la letra sin sincronizar

#### Scenario: Normalized query used

- GIVEN un título sucio de MV (`"Artist - Song (Official Music Video)"`, uploader `"ArtistVEVO"`)
- WHEN se solicita la letra y no hay referencia guardada
- THEN la consulta usa el `(artist, title)` normalizado, no los campos crudos

#### Scenario: Search fallback after get miss

- GIVEN `/api/get` no devuelve coincidencia para la consulta normalizada
- WHEN aún se está resolviendo la letra
- THEN el sistema reintenta con `/api/search` del proveedor usando la misma consulta normalizada
- AND se devuelve la letra si `/api/search` encuentra un candidato

## ADDED Requirements

### Requirement: Manual Lyrics Search

The system MUST allow the user to search for lyrics manually by entering a free-text
query for the current track. The system MUST query the provider's fuzzy `/api/search`
endpoint and present the ranked candidates (with at least track name and artist) so the
user can select one. Selecting a candidate MUST load and display its lyrics, preferring
synced over plain.

#### Scenario: Manual search returns candidates

- GIVEN suena una pista y el usuario abre la búsqueda manual de letra
- WHEN teclea una consulta y la envía
- THEN se muestran los candidatos de `/api/search` con nombre de pista y artista

#### Scenario: Select a candidate

- GIVEN hay candidatos de letra mostrados
- WHEN el usuario elige uno
- THEN se carga y muestra su letra (sincronizada si está disponible)

#### Scenario: Manual search with no candidates

- GIVEN el usuario envía una consulta de letra
- WHEN el proveedor no devuelve candidatos
- THEN se informa "sin resultados" y la reproducción continúa normal

### Requirement: Persist Lyrics Reference

When the user selects a lyrics candidate via manual search, the system MUST persist the
query and/or provider reference (e.g., the provider track id) linked to the track, so a
future playback of the same track reuses it. The persisted reference MUST survive across
sessions and MUST be keyed by the track's video id.

#### Scenario: Reference saved on selection

- GIVEN el usuario elige un candidato de letra para una pista
- WHEN se confirma la selección
- THEN se guarda la consulta y/o la referencia del proveedor vinculada al `video_id`

#### Scenario: Reference reused on replay

- GIVEN una pista con referencia de letra guardada
- WHEN se vuelve a reproducir en otra sesión
- THEN la letra se resuelve directamente con la referencia guardada sin requerir una nueva búsqueda manual
