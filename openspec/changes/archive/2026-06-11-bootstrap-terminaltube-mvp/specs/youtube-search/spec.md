# Delta for youtube-search

## ADDED Requirements

### Requirement: Search by Free Text

The system MUST accept a free-text query and return up to N results (default 10)
via `yt-dlp "ytsearchN:<query>" --dump-json --flat-playlist`.

#### Scenario: Successful search

- GIVEN yt-dlp en PATH
- WHEN el usuario busca un término
- THEN se devuelven resultados con id, título, autor y duración (≤ N)

### Requirement: Structured Results

Each result MUST expose id, title, uploader and duration; entries without id are
discarded.

#### Scenario: Skip malformed entries

- GIVEN una línea JSON sin id
- WHEN se parsea
- THEN se descarta y el resto se conserva

### Requirement: Graceful Failure

The system MUST surface a clear error if yt-dlp is missing or fails, without crashing.

#### Scenario: yt-dlp missing

- GIVEN yt-dlp no está en PATH
- WHEN se busca
- THEN error legible y la TUI sigue operativa
