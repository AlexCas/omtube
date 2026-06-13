# YouTube Search Specification

## Purpose

Buscar canciones en YouTube usando `yt-dlp` y exponer resultados estructurados a la
UI, sin descargar audio ni depender de APIs oficiales.

## Requirements

### Requirement: Search by Free Text

The system MUST accept a free-text query and return up to N results (configurable,
default 10) using `yt-dlp "ytsearchN:<query>" --dump-json --flat-playlist`.

#### Scenario: Successful search

- GIVEN yt-dlp está disponible en PATH
- WHEN el usuario busca "Linkin Park Numb"
- THEN se devuelve una lista de resultados con id, título, autor y duración
- AND la lista no excede N elementos

#### Scenario: Empty query

- GIVEN el campo de búsqueda está vacío o solo con espacios
- WHEN el usuario envía la búsqueda
- THEN no se ejecuta yt-dlp y no se modifican los resultados

### Requirement: Structured Results

Each result MUST expose at least video id, title, uploader, and duration (seconds).
A result with a missing or empty id MUST be discarded.

#### Scenario: Skip malformed entries

- GIVEN yt-dlp emite una línea JSON sin campo id
- WHEN se parsea la salida
- THEN esa entrada se descarta y el resto se conserva

### Requirement: Graceful Failure

The system MUST surface a clear error when yt-dlp is missing or exits non-zero,
without crashing the app.

#### Scenario: yt-dlp missing

- GIVEN yt-dlp no está en PATH
- WHEN se intenta buscar
- THEN se devuelve un error legible indicando la dependencia faltante
- AND la TUI sigue operativa

### Requirement: Resolve a Video URL

The system MUST accept a single YouTube video URL, classify it as a video, and resolve
it to one structured result (id, title, uploader, duration) using `yt-dlp --dump-json`.
A `watch?v=<id>` URL that also carries a `list=` parameter MUST be treated as a single
video, not a playlist.

#### Scenario: Resolve a watch URL

- GIVEN una URL `https://www.youtube.com/watch?v=ID`
- WHEN el usuario la introduce
- THEN se resuelve a un único resultado con id, título, autor y duración

#### Scenario: Resolve a short youtu.be URL

- GIVEN una URL `https://youtu.be/ID`
- WHEN el usuario la introduce
- THEN se extrae el id y se resuelve a un único resultado

#### Scenario: Watch URL with list parameter

- GIVEN una URL `https://www.youtube.com/watch?v=ID&list=PL123`
- WHEN el usuario la introduce como vídeo
- THEN se resuelve solo el vídeo `ID`, ignorando `list=`

#### Scenario: Unresolvable URL

- GIVEN una URL que yt-dlp no puede resolver o no es de YouTube
- WHEN se intenta resolver
- THEN se devuelve un error legible y la TUI sigue operativa

### Requirement: Resolve a Playlist URL

The system MUST accept a YouTube playlist URL, classify it as a playlist, and resolve it
to an ordered list of results plus the playlist title using
`yt-dlp --flat-playlist --dump-json`. Entries without an id MUST be discarded.

#### Scenario: Resolve a playlist URL

- GIVEN una URL `https://www.youtube.com/playlist?list=PL123`
- WHEN el usuario la importa
- THEN se devuelven sus entradas en orden, cada una con id y título, y el título de la playlist

#### Scenario: Skip malformed playlist entries

- GIVEN yt-dlp emite una entrada de playlist sin id
- WHEN se parsea la salida
- THEN esa entrada se descarta y el resto se conserva en orden

#### Scenario: Empty or private playlist

- GIVEN una URL de playlist sin entradas accesibles
- WHEN se intenta resolver
- THEN se devuelve una lista vacía o un error legible sin bloquear la TUI
