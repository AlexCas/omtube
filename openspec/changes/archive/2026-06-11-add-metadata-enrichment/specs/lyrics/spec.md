# Delta for Lyrics

## MODIFIED Requirements

### Requirement: Fetch Lyrics

The system MUST fetch lyrics for the current track from a community API using a
normalized `(artist, title)` query rather than the raw YouTube title/uploader,
preferring synced (.lrc) lyrics and falling back to plain text. On a strict
`/api/get` miss, the system MUST retry the same provider's fuzzy `/api/search` endpoint
with the same normalized query before declaring "no lyrics". Fetching MUST be
controllable by a config toggle, and results SHOULD be cached.
(Previously: queried lrclib `/api/get` only, with the raw title/uploader and no fuzzy fallback.)

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
- WHEN se solicita la letra
- THEN la consulta usa el `(artist, title)` normalizado, no los campos crudos

#### Scenario: Search fallback after get miss

- GIVEN `/api/get` no devuelve coincidencia para la consulta normalizada
- WHEN aún se está resolviendo la letra
- THEN el sistema reintenta con `/api/search` del proveedor usando la misma consulta normalizada
- AND se devuelve la letra si `/api/search` encuentra un candidato
