# Tasks: Ingesta por URL y memoria de letra

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~900-1200 (2 nuevos archivos + ~13 modificados + migración + tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (search+queue+player) → PR 2 (storage+lyrics) → PR 3 (UI+wiring+docs) |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain (3 PRs) |

Decision needed before apply: Yes (teclas y alcance de referencia de letra — ver design.md "Open Decisions")
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units (3-PR split)

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | `search` (clasificar URL + Resolve/ResolvePlaylist), `queue.Clear`, `player.Stop` + tests | PR 1 | Paquetes hoja; compila y tests verdes en solitario. ~300-400 líneas. |
| 2 | Migración 3 + `lyrics` (Search/SelectCandidate/reuso) + storage + tests | PR 2 | Base = PR 1. Aditivo; tests verdes en solitario. ~300-400 líneas. |
| 3 | UI (modos/teclas/mensajes), wiring, README + tests | PR 3 | Base = PR 2. Integra todo. ~300-400 líneas. |

Cada PR construye y `go test ./...` en verde antes del siguiente.

## Phase 1: Search + Queue + Player (PR 1)

- [x] 1.1 Create `internal/search/url.go`: `URLKind` (Unknown/Video/Playlist) y
  `ClassifyURL(raw)→(URLKind, id)` con `net/url`. Regla: `watch?v=` y `youtu.be/<id>` y
  `/shorts/<id>` ⇒ Video (ignora `list=`); `/playlist?list=` o solo `list=` ⇒ Playlist.
- [x] 1.2 Modify `internal/search/ytdlp.go`: `Resolve(ctx,url)` (`--dump-json` sin
  `--flat-playlist`, primer resultado con id) y `ResolvePlaylist(ctx,url)`
  (`--flat-playlist --dump-json`, `[]Result` en orden + título de playlist). Reusar
  `parseEntries`; error legible si yt-dlp falla.
- [x] 1.3 Modify `internal/search/search.go`: declarar interfaces `Resolver` /
  `PlaylistResolver` (o ampliar `Searcher`) para que la UI dependa de la abstracción.
- [x] 1.4 Modify `internal/queue/queue.go`: `Clear()` (items=nil; idx=-1); zero-value
  sigue usable y un `Add` posterior vuelve a fijar la pista actual.
- [x] 1.5 Modify `internal/player/player.go`: añadir `Stop() error` a la interfaz `Player`.
- [x] 1.6 Modify `internal/player/mpv.go`: implementar `Stop()` enviando el comando `stop`
  a mpv (deja el proceso vivo y ocioso, sin matarlo como `Close`).

## Phase 2: Storage + Lyrics Memory (PR 2)

- [x] 2.1 Modify `internal/storage/migrate.go`: añadir `migrations[2]` (migración 3) con
  `ALTER TABLE lyrics_cache ADD COLUMN query TEXT NOT NULL DEFAULT ''` y
  `ADD COLUMN provider_id TEXT NOT NULL DEFAULT ''`.
- [x] 2.2 Modify `internal/storage/lyrics_cache.go`: añadir `Query` y `ProviderID` a
  `LyricsEntry`; persistirlos en `upsertLyricsQuery`/`UpsertWithTrack`; leerlos en `Get`.
- [x] 2.3 Modify `internal/lyrics/lyrics.go`: `Candidate` struct; `Search(ctx,query)` que
  consulta `/api/search` y devuelve candidatos rankeados (reusar `pickBestCandidate`/
  parsing existente, exponiendo id de proveedor); `SelectCandidate(ctx,track,c)` que carga
  la letra del candidato y persiste `query`/`provider_id` vía `UpsertWithTrack`.
- [x] 2.4 Modify `internal/lyrics/lyrics.go`: en `Fetch`, si `Get(track.ID)` trae
  `provider_id`/`query` guardados, resolver con esa referencia (`/api/get?...` por id o la
  query guardada) antes de la consulta normalizada automática.

## Phase 3: UI + Wiring + Docs (PR 3)

- [x] 3.1 Modify `internal/ui/keys.go`: teclas `AddFromURL`, `ImportPlaylist`,
  `LyricsSearch`, `ClearQueue` (propuesta: `u`, `i`, `y`, `C`).
- [x] 3.2 Modify `internal/ui/model.go`: modos `modeURLInput`, `modeImportURL`,
  `modeImportName`, `modeLyricsSearch`; estado intermedio (tracks importadas pendientes de
  nombre, candidatos de letra) y reuso de `input`/`picker`.
- [x] 3.3 Modify `internal/ui/messages.go`: `resolveURLCmd`/`urlResolvedMsg`,
  `resolvePlaylistCmd`/`playlistResolvedMsg`, `lyricsSearchCmd`/`lyricsCandidatesMsg`,
  `selectLyricsCmd` (reusa `lyricsMsg`).
- [x] 3.4 Modify `internal/ui/update.go`: handlers de los nuevos modos; en URL resuelta:
  `queue.Add` + `results=[track]` + arranque si `!started`; import bifásico (URL→nombre→
  `Create`+`Add`*); selección de candidato persiste y actualiza panel; `ClearQueue`:
  `queue.Clear()`+`player.Stop()`+reset `curTrackID`/`curLyrics`/`curArtwork`/`started`.
- [x] 3.5 Modify `internal/ui/view.go`: render de los nuevos modos (prompts/listas) y
  actualizar la línea de ayuda de teclas.
- [x] 3.6 Modify `main.go` si hace falta: nada de config nuevo; verificar wiring de
  `Resolver`/`PlaylistResolver` hacia `ui.New`.
- [x] 3.7 Modify `README.md`: documentar teclas y flujos (añadir por URL, importar
  playlist, buscar letra, limpiar cola).

## Phase 4: Testing

- [x] 4.1 Create `internal/search/url_test.go`: tabla por formato de URL (watch, youtu.be,
  shorts, playlist, watch+list, no-YouTube).
- [x] 4.2 Modify `internal/search/ytdlp_test.go`: `Resolve`/`ResolvePlaylist` con fake
  yt-dlp; descarte de entradas sin id; orden preservado.
- [x] 4.3 Modify `internal/queue/queue_test.go`: `Clear` vacía y resetea idx; re-`Add` fija
  la pista actual.
- [x] 4.4 Modify `internal/storage/migrate_test.go`: migración 3 idempotente; `user_version`
  2→3; filas previas conservan default en columnas nuevas.
- [x] 4.5 Modify `internal/lyrics/lyrics_test.go` (`httptest.Server`): `Search` candidatos;
  `SelectCandidate` carga+persiste `query`/`provider_id`; `Fetch` reusa la referencia
  guardada antes que la consulta automática.
- [x] 4.6 Modify `internal/ui/update_test.go`: `Update` de modos nuevos (URL/import/letra) y
  `ClearQueue` (cola vacía, panels reseteados).

## Verification

- [x] `go build ./...` en verde.
- [x] `go test ./...` en verde.
- [ ] Smoke manual de TUI: URL→cola(+playlist), import→playlist, letra manual→persiste/
  reusa, limpiar→vacía+para.
