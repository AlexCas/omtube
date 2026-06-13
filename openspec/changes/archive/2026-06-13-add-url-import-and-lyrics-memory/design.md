# Design: Ingesta por URL y memoria de letra

## Technical Approach

Cuatro capacidades, todas extendiendo paquetes existentes sin dependencias nuevas:

1. **URL → resultado(s)**: nuevo `internal/search/url.go` clasifica una URL de YouTube
   (`Kind`: vídeo / playlist / desconocida) y extrae el id de vídeo o de lista. Regla de
   desambiguación: `watch?v=` (aunque traiga `list=`) y `youtu.be/<id>` y `/shorts/<id>`
   ⇒ **vídeo**; `/playlist?list=` o una URL con solo `list=` ⇒ **playlist**. `YtDlp` gana
   `Resolve(ctx, url)` (vídeo único, `--dump-json` sin `--flat-playlist` para metadato
   completo) y `ResolvePlaylist(ctx, url)` (`--flat-playlist --dump-json`, devuelve
   `[]Result` + título). Ambos reutilizan `parseEntries`.
2. **Limpiar cola**: `queue.Clear()` (items=nil, idx=-1) + nuevo `player.Stop()` (comando
   `stop` de mpv). La UI resetea el estado de "ahora suena" (letra/portada/`started`).
3. **Memoria de letra**: migración 3 añade `query` y `provider_id` a `lyrics_cache`. El
   `lyrics.Service` gana `Search(ctx, query)` (candidatos de `/api/search` con su id y
   metadato) y `SelectCandidate(ctx, track, cand)` (carga letra del candidato + persiste
   `query`/`provider_id` vía `UpsertWithTrack`). `Fetch` consulta primero la referencia
   guardada (`provider_id`→`/api/get` por id, o `query` guardada) antes de la consulta
   automática.
4. **UI**: tres modos nuevos (`modeURLInput`, `modeImportURL`, `modeLyricsSearch`) que
   reutilizan el `textinput` compartido, más reuso del `picker` (`list.Model`) para los
   candidatos de letra. La importación es bifásica (URL → nombre) con estado intermedio
   en el `Model`. Cada operación remota es un `tea.Cmd` async con su mensaje.

Todo degrada limpio: yt-dlp/lrclib que fallan ⇒ error legible en barra de estado, la TUI
sigue operativa; la migración es aditiva.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Clasificar URL | Parser propio en `search/url.go` con `net/url` | delegar todo a yt-dlp | Decide vídeo vs playlist sin lanzar proceso; testeable por tabla |
| `watch?v=`+`list=` | Tratar como vídeo | preguntar al usuario | Coincide con la intención de "añadir esta canción"; importar usa `/playlist` |
| Resolver vídeo | `Resolve` sin `--flat-playlist` | con flat | Metadato completo (duración/uploader) para una sola pista |
| Resolver playlist | `ResolvePlaylist` con `--flat-playlist` | resolver cada vídeo | Rápido; no descarga; orden preservado |
| Parar reproducción | Nuevo `player.Stop()` (mpv `stop`) | `Close()`+recrear; pausar | `Close` mata mpv; pausar no vacía; `stop` deja mpv vivo y ocioso |
| Reuso de cola tras Clear | `Add` vuelve a fijar actual e índice | recrear `Queue` | `Clear` deja el zero-value útil; sin punteros colgantes en la UI |
| Memoria de letra | Columnas `query`+`provider_id` en `lyrics_cache` | tabla nueva; en history | Misma clave `video_id`; reusa `LyricsRepo`/FK; mínima superficie |
| Referencia preferida | `provider_id` (id de pista lrclib) | solo query | Trae exactamente la misma letra; `query` es respaldo legible |
| Selección manual | Reusa `picker` (`list.Model`) | nuevo componente | Mismo patrón que "añadir a playlist"; sin UI nueva |
| Importar nombre | Tecleado por el usuario (decisión de producto) | título de YouTube | Evita nombres sucios/duplicados; control explícito |
| Letra: prefill | Query manual prellenada con `(artist,title)` normalizado | vacío | Menos tecleo; el usuario solo ajusta |

## Data Flow

    [Añadir por URL]
    modeURLInput (enter) ─▶ resolveURLCmd(url) ─▶ urlResolvedMsg{track,err}
       └─ ok: queue.Add(track); results=[track]; if !started→LoadTrack
              status "encolado · a → playlist"  (la tecla 'a' abre el picker existente)

    [Importar playlist]
    modeImportURL (enter) ─▶ resolvePlaylistCmd(url) ─▶ playlistResolvedMsg{tracks,title,err}
       └─ ok: stash tracks; modeImportName (textinput) ─▶ (enter)
              playlists.Create(name) → for t: playlists.Add(id,t) → status

    [Búsqueda manual de letra]
    modeLyricsSearch (enter) ─▶ lyricsSearchCmd(query) ─▶ lyricsCandidatesMsg{cands,err}
       └─ picker(cands) ─▶ (enter) selectLyricsCmd(track,cand) ─▶ lyricsMsg + persist(query,provider_id)

    [Limpiar cola]
    tecla ClearQueue ─▶ queue.Clear(); player.Stop(); reset curTrackID/curLyrics/curArtwork/started

    [Reuso en re-reproducción]
    EventTrackChange ─▶ fetchLyricsCmd ─▶ lyrics.Fetch:
       saved ref? sí→/api/get?id=provider_id (o query guardada) ; no→consulta normalizada

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/search/url.go` | Create | `ClassifyURL(raw)→(Kind, id string)`; `Kind`: Video/Playlist/Unknown |
| `internal/search/url_test.go` | Create | Tabla por formato (watch, youtu.be, shorts, playlist, watch+list) |
| `internal/search/ytdlp.go` | Modify | `Resolve(ctx,url)→Result`; `ResolvePlaylist(ctx,url)→([]Result,string,error)` |
| `internal/search/ytdlp_test.go` | Modify | Tests con fake yt-dlp para resolve/resolvePlaylist |
| `internal/search/search.go` | Modify | Ampliar `Searcher` o añadir interfaces `Resolver`/`PlaylistResolver` |
| `internal/queue/queue.go` | Modify | `Clear()` |
| `internal/queue/queue_test.go` | Modify | Clear vacía y resetea idx; re-Add fija actual |
| `internal/player/player.go` | Modify | Añadir `Stop() error` a la interfaz |
| `internal/player/mpv.go` | Modify | `Stop()` envía comando `stop` a mpv |
| `internal/storage/migrate.go` | Modify | Migración 3: `ALTER TABLE lyrics_cache ADD COLUMN query/provider_id` |
| `internal/storage/lyrics_cache.go` | Modify | Persistir/leer `query`,`provider_id`; `LyricsEntry` + campos |
| `internal/storage/migrate_test.go` | Modify | Migración 3 idempotente; user_version 2→3 |
| `internal/lyrics/lyrics.go` | Modify | `Search`, `SelectCandidate`, reuso de ref guardada en `Fetch` |
| `internal/lyrics/lyrics_test.go` | Modify | Candidatos, selección+persistencia, reuso de ref (httptest) |
| `internal/ui/keys.go` | Modify | Teclas: AddFromURL, ImportPlaylist, LyricsSearch, ClearQueue |
| `internal/ui/messages.go` | Modify | Cmds/Msgs: resolveURL, resolvePlaylist, lyricsSearch, selectLyrics |
| `internal/ui/model.go` | Modify | Nuevos modos + estado intermedio (tracks importadas, candidatos) |
| `internal/ui/update.go` | Modify | Handlers de los nuevos modos; reset en ClearQueue |
| `internal/ui/view.go` | Modify | Render de los nuevos modos y ayuda de teclas |
| `internal/ui/update_test.go` | Modify | Update de modos nuevos y ClearQueue |
| `README.md` | Modify | Documentar teclas y flujos nuevos |

## Interfaces / Contracts

```go
// search
type URLKind int
const ( URLUnknown URLKind = iota; URLVideo; URLPlaylist )
func ClassifyURL(raw string) (kind URLKind, id string)        // id = video id o list id
func (y *YtDlp) Resolve(ctx context.Context, url string) (Result, error)
func (y *YtDlp) ResolvePlaylist(ctx context.Context, url string) (tracks []Result, title string, err error)

// queue
func (q *Queue) Clear()                                       // items=nil; idx=-1

// player
Stop() error                                                  // mpv "stop"; deja mpv vivo

// lyrics
type Candidate struct { ProviderID string; Title, Artist string; Duration int; Synced bool }
func (s *Service) Search(ctx context.Context, query string) ([]Candidate, error)        // /api/search
func (s *Service) SelectCandidate(ctx context.Context, track search.Result, c Candidate) (Lyrics, error) // carga + persiste query/provider_id
// Fetch: si hay (query/provider_id) guardado para track.ID, lo usa; si no, consulta normalizada
func (s *Service) Fetch(ctx context.Context, track search.Result, queryTitle, queryArtist string) (Lyrics, error)

// storage
type LyricsEntry struct { VideoID string; Synced bool; Body, FetchedAt, Query, ProviderID string }
```

Migración 3 (aditiva, en `migrations[2]`):
`ALTER TABLE lyrics_cache ADD COLUMN query TEXT NOT NULL DEFAULT '';`
`ALTER TABLE lyrics_cache ADD COLUMN provider_id TEXT NOT NULL DEFAULT '';`
(SQLite ejecuta múltiples sentencias en la misma transacción de migración.)

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `ClassifyURL` por formato (watch, youtu.be, shorts, playlist, watch+list, no-YT) | tabla |
| Unit | `Resolve`/`ResolvePlaylist` parsean NDJSON; descartan entradas sin id | fake yt-dlp script en `t.TempDir()` |
| Unit | `queue.Clear` vacía+resetea; re-`Add` fija actual | tabla |
| Unit | `lyrics.Search` candidatos; `SelectCandidate` carga+persiste; `Fetch` reusa ref guardada antes que auto-query | `httptest.Server` + repo en tmp DB |
| Integration | migración 3 idempotente; `user_version` 2→3; columnas con default no rompen filas previas | abrir DB dos veces |
| Unit | `Update` de modos nuevos (URL/import/letra) y reset en ClearQueue | teatest/golden donde aplique |
| E2E | pegar URL encola; importar crea playlist; letra manual persiste y se reusa; limpiar vacía+para | smoke manual de TUI |

## Migration / Rollout

Migración 3 solo AÑADE columnas con `DEFAULT ''` → bibliotecas existentes intactas; un
binario anterior ignora las columnas. Sin flags de config nuevos (las features son
acciones de UI, siempre disponibles). Rollback = revertir el PR; los datos persistidos
siguen siendo legibles.

## Resolved Decisions (human review gate)

- [x] **Añadir por URL**: auto-encola y además expone la pista como resultado único para
  que la tecla de "añadir a playlist" funcione sobre ella (reuso del picker).
- [x] **Limpiar cola**: limpiar todo y parar (no conservar la actual).
- [x] **Letra**: búsqueda manual con lista de candidatos (no solo persistir el autofetch).
- [x] **Nombre al importar**: lo teclea el usuario (no se usa el título de YouTube).

## Open Decisions (for review before apply)

- [ ] **Teclas concretas**: propongo `u` (añadir por URL), `i` (importar playlist),
  `y` (buscar letra), `C`/`shift+c` (limpiar cola). ¿Confirmas o ajustas?
- [ ] **Referencia de letra**: persistir `provider_id` **y** `query`; al reusar, intentar
  `provider_id` primero y caer a `query`. ¿Suficiente, o solo `query`?
