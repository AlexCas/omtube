# Delta for youtube-search

## ADDED Requirements

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
