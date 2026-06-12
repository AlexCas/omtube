# Delta for audio-playback

## MODIFIED Requirements

### Requirement: Load and Transport Control

The system MUST load a track via `loadfile` and control pause and volume via
`set_property`. When a valid cached local file exists for the track, the system MUST
load that local file instead of resolving/streaming the YouTube id.
(Previously: siempre cargaba por id de YouTube; ahora prioriza el archivo cacheado si
existe.)

#### Scenario: Play a track

- GIVEN el reproductor está activo y la pista no está cacheada
- WHEN se carga el id de un video
- THEN mpv reproduce su audio (resuelto por su hook yt-dlp, formato bestaudio)

#### Scenario: Play a cached track

- GIVEN existe un archivo local válido en caché para la pista
- WHEN se carga esa pista
- THEN mpv reproduce el archivo local sin resolver el id de YouTube

#### Scenario: Toggle pause and volume

- GIVEN una pista en reproducción
- WHEN el usuario alterna pausa o ajusta volumen
- THEN mpv aplica el cambio vía set_property
- AND el volumen se mantiene en el rango 0–130

### Requirement: Progress and End Events

The system MUST expose current position/duration and emit an event when a track ends,
over an event channel. The system MUST also emit a track-change event when a new track
starts loading, so subscribers (lyrics, artwork, Discord presence) can react.
(Previously: solo exponía posición/duración y evento de fin; ahora también emite evento
de cambio de pista.)

#### Scenario: Track ends

- GIVEN una pista llega a su fin
- WHEN mpv emite `end-file`
- THEN se emite un evento de fin por el canal para que la cola avance

#### Scenario: Track changes

- GIVEN una nueva pista comienza a cargarse
- WHEN se inicia su reproducción
- THEN se emite un evento de cambio de pista con su metadato por el canal
