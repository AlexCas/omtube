# Delta for Artwork

## MODIFIED Requirements

### Requirement: Render Current Track Artwork

The system MUST fetch and display artwork for the current track and MUST update it when
the track changes. When the cover-art toggle is on, the system MUST attempt to resolve
the real release cover via MusicBrainz recording/release search followed by the Cover
Art Archive front cover, keyed off the normalized `(artist, title)`. Cover-art lookups
MUST send a descriptive User-Agent, MUST be throttled to roughly one request per second,
and MUST cache results including negative (no-match) outcomes. On any miss, offline
state, or when the toggle is off, the system MUST fall back to the YouTube thumbnail.
(Previously: artwork was always the YouTube thumbnail — cached `--write-thumbnail` file or `i.ytimg` `hqdefault.jpg` — with no release-cover source.)

#### Scenario: Show artwork on play

- GIVEN el toggle de portada está activo y la terminal es capaz
- WHEN arranca una pista con thumbnail disponible
- THEN se muestra su portada en el panel correspondiente

#### Scenario: Real cover resolved

- GIVEN el toggle de portada está activo y MusicBrainz + Cover Art Archive devuelven una portada frontal para el `(artist, title)` normalizado
- WHEN suena la pista
- THEN se obtiene, cachea y renderiza la portada real de la release en lugar del thumbnail

#### Scenario: Cover lookup cached

- GIVEN ya se realizó una búsqueda de portada (positiva o negativa) para una pista
- WHEN vuelve a sonar la misma pista
- THEN se usa el resultado cacheado y no se hace una nueva petición a MusicBrainz/Cover Art Archive

#### Scenario: Update on track change

- GIVEN se está mostrando la portada de la pista actual
- WHEN la reproducción avanza a la siguiente pista
- THEN la portada se actualiza a la de la nueva pista

#### Scenario: Thumbnail fallback on miss or offline

- GIVEN el toggle de portada está apagado, o la búsqueda falla, o no hay red
- WHEN se solicita la portada
- THEN el sistema recurre al thumbnail de YouTube sin error

#### Scenario: Artwork unavailable

- GIVEN una pista sin thumbnail o cuya descarga falla
- WHEN se solicita la portada
- THEN el panel queda vacío/placeholder sin interrumpir la reproducción
