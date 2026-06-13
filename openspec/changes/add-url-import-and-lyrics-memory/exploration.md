## Exploration: Ingesta por URL, importación de playlists, memoria de letra y limpiar cola

### Current State

- **Búsqueda**: `search.Searcher` solo acepta texto libre (`ytsearchN:<q>`). No hay
  parseo de URLs ni resolución de un vídeo/playlist concreto. `parseEntries`
  (`internal/search/ytdlp.go`) ya convierte NDJSON de yt-dlp en `[]search.Result`
  y es reutilizable para resolver una URL única o una playlist.
- **Cola**: `queue.Queue` (`internal/queue/queue.go`) es append-only: `Add`, `Next`,
  `Prev`, `Current`, `Items`, `Index`, `Len`. **No** existe `Clear`/`Remove`.
- **Player**: `player.Player` (`internal/player/player.go`) expone `Load`,
  `LoadTrack`, `TogglePause`, `AddVolume`, `Position`, `Paused`, `Volume`,
  `Events`, `Close`. **No** existe `Stop()` — necesario para "limpiar y parar".
- **Playlists**: `playlist.Service` ofrece `Create`, `Add(id, track)`, `Tracks`,
  `Play`, etc. Son listas locales en SQLite (`playlists`/`playlist_tracks`), no
  playlists de YouTube. `Add` hace upsert del track antes de insertar membresía.
- **Letra**: `lyrics.Service.Fetch(ctx, track, queryTitle, queryArtist)` consulta
  lrclib `/api/get` y, como fallback, `/api/search` (con `pickBestCandidate`).
  Cachea por `video_id` en `lyrics_cache(video_id, synced, body, fetched_at)`. La
  consulta usada se calcula on-the-fly a partir de título/artista normalizados;
  **no se persiste** la query ni la referencia del proveedor, así que un título
  sucio (típico al pegar una URL) no es corregible ni memorizable.
- **UI**: Bubble Tea con `mode` enum (`modeNormal`, `modeSearch`, `modeLibrary`,
  `modePicker`, `modeCreatePlaylist`). Un único `textinput` compartido y un
  `list.Model` (`picker`) para "añadir a playlist". Flujo async vía `tea.Cmd` +
  mensajes (`searchResultsMsg`, `loadedMsg`, etc.). El picker ya implementa
  "añadir a playlist existente" sobre un `search.Result`.

### Affected Areas

- `internal/search/ytdlp.go` — nuevos `Resolve` (URL única) y `ResolvePlaylist`.
- `internal/search/url.go` (nuevo) — parseo/clasificación de URLs de YouTube.
- `internal/queue/queue.go` — `Clear()`.
- `internal/player/player.go` + `mpv.go` — `Stop()`.
- `internal/lyrics/lyrics.go` — búsqueda manual de candidatos + persistencia de
  query/referencia y reuso en re-reproducción.
- `internal/storage/lyrics_cache.go` + `migrate.go` — migración 3: columnas
  `query`, `provider_id`.
- `internal/ui/*` — modos nuevos (URL, importar playlist, búsqueda de letra),
  teclas y mensajes async.

### Approaches

1. **Extender paquetes existentes (recomendado)** — añadir métodos a `search`,
   `queue`, `player`, `lyrics`; nuevos modos en la UI reusando `textinput` y
   `picker`. Migración 3 aditiva en `lyrics_cache`.
   - Pros: sigue patrones de las fases previas; cambios localizados; sin deps nuevas.
   - Cons: la UI gana varios modos (gestión de estado intermedia para 2 pasos).
   - Effort: Medium.

2. **Paquete `ingest` nuevo que orqueste URL→cola/playlist** — capa separada.
   - Pros: separa la lógica de ingesta.
   - Cons: duplica responsabilidades de `search`/`playlist`; sobre-ingeniería para
     el tamaño del proyecto.
   - Effort: High.

### Recommendation

Enfoque 1. La resolución de URL/playlist es una variante de lo que `yt-dlp` ya hace
en `search`; la cola, el player y las letras solo necesitan métodos puntuales; y la
UI ya tiene los componentes (input + picker) para los flujos nuevos.

### Risks

- yt-dlp puede tardar/fallar al resolver URLs o playlists grandes → timeouts y
  degradación clara.
- Variedad de formatos de URL de YouTube (watch, youtu.be, shorts, con `list=`).
- La memoria de letra cambia el esquema (migración 3): debe ser idempotente y
  aditiva para no romper bibliotecas existentes.

### Ready for Proposal

Yes — las 4 decisiones de producto están resueltas (auto-encolar + opción playlist;
búsqueda manual de letra con candidatos; limpiar todo y parar; nombre de playlist
tecleado por el usuario).
