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
