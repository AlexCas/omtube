# Tasks: Media Enrichment (Fase 3)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1100-1400 (8 new pkgs/files + 8 modified + migration + tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (cache+storage) → PR 2 (lyrics+artwork+presence) → PR 3 (player+UI+main wiring) |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain (3 PRs) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units (3-PR split)

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Storage migration 2 + `internal/cache` (index, download, eviction, startup sweep) + unit tests | PR 1 | Base = feature/media-enrichment. Only existing-file change is `storage/*`. Compiles + tests green standalone. ~400-500 lines. |
| 2 | `internal/lyrics`, `internal/artwork`, `internal/presence` + `rich-go` dep + unit tests | PR 2 | Base = PR 1 branch. Pure leaf packages, no UI/player edits yet. Tests green standalone. ~400-500 lines. |
| 3 | `internal/player` events + cache-aware Load, `internal/config` toggles, `internal/ui` panels/wiring, `main.go`, teatest/golden, docs | PR 3 | Base = PR 2 branch. Integrates all packages. ~300-400 lines. |

Each PR builds and `go test ./...` green independently before the next. PR #1 targets the tracker branch; #2 targets #1; #3 targets #2 (retarget/rebase if a child diff shows parent changes).

## Phase 1: Storage + Cache (PR 1)

- [x] 1.1 Modify `internal/storage/migrate.go`: append migration 2 — `cache_entries(video_id PK, path, size_bytes, ext, created_at, last_used)` FK→`tracks` ON DELETE CASCADE; `lyrics_cache(video_id PK, synced INT, body TEXT, fetched_at)` FK→`tracks`; bumps `user_version` 1→2.
- [x] 1.2 Modify `internal/storage/storage.go`: add `Cache()` and `Lyrics()` repo accessors.
- [x] 1.3 Create `internal/storage/cache_entries.go`: `CacheRepo` Upsert/Get/Delete/List(by last_used)/TotalBytes over `cache_entries`.
- [x] 1.4 Create `internal/storage/lyrics_cache.go`: `LyricsRepo` Upsert/Get over `lyrics_cache`.
- [x] 1.5 Create `internal/cache/index.go`: thin index ops bridging `CacheRepo` (`record`, `touch`, `oldest`, `total`).
- [x] 1.6 Create `internal/cache/cache.go`: `New(repo, ytdlp, dir, maxBytes, maxAge)`; `Lookup(id)→(path,ok)` (validates file exists/non-empty, else invalidates row); `Download(ctx,r)` via `yt-dlp -x --write-thumbnail` to `<dir>/audio/<id>.<ext>`; `Evict()` (size+age, delete oldest by last_used); `Sweep()` startup eviction; `Clear()`.

## Phase 2: Lyrics, Artwork, Presence (PR 2)

- [x] 2.1 Add `github.com/hugolgst/rich-go` to `go.mod`/`go.sum` (`go get`, `go mod tidy`).
- [x] 2.2 Create `internal/lyrics/lrc.go`: `.lrc` parser → `Lyrics{Synced bool; Lines []Line{T,Text}; Plain string}`; `(Lyrics) LineAt(sec float64) int` (binary search).
- [x] 2.3 Create `internal/lyrics/lyrics.go`: `New(repo, httpClient)`; `Fetch(ctx,title,artist,dur)` to lrclib, prefer synced, DB-cache hit skips HTTP; failure/no-match ⇒ empty `Lyrics`, no error surfaced.
- [x] 2.4 Create `internal/artwork/artwork.go`: `Detect()→Backend` (kitty→sixel→chafa→none via `$TERM`/`$KITTY_WINDOW_ID`/`$TERM_PROGRAM`); `(Backend) Render(ctx,thumbURL,w,h)` → escape seq/placeholder; unavailable ⇒ placeholder, no error.
- [x] 2.5 Create `internal/presence/presence.go`: `New(appID)`; `Connect()` silent no-op if `appID==""` (log once) or IPC fails; `Set(title,artist)`, `Clear()`, `Close()`.

## Phase 3: Player + Config + UI + Wiring (PR 3)

- [x] 3.1 Modify `internal/player/player.go`: add `EventTrackChange` kind; `Event` carries `Track search.Result` + `Source string`.
- [x] 3.2 Modify `internal/player/mpv.go`: `Load(src)` loads local file path or YouTube id/URL; emit `EventTrackChange` on new track.
- [x] 3.3 Modify `internal/config/config.go`: add `cache.enabled/max_size_mb/max_age_days`, `lyrics.enabled`, `artwork.enabled`, `presence.enabled`, `presence.app_id`; add `CacheDir()` (XDG). Presence stays off unless `app_id` set.
- [x] 3.4 Modify `internal/ui/messages.go`: add `lyricsMsg`/`artworkMsg`; `fetchLyricsCmd`, `renderArtworkCmd`, `setPresenceCmd`, `cacheDownloadCmd`.
- [x] 3.5 Modify `internal/ui/model.go`: hold cache/lyrics/artwork/presence services + lyrics/artwork panel state + per-track cached flags.
- [x] 3.6 Modify `internal/ui/update.go`: cache `Lookup` before `Load`; on `EventTrackChange` fan out Cmds (lyrics/artwork/presence/cache-download); advance lyric highlight on `posMsg`/`tickMsg`.
- [x] 3.7 Modify `internal/ui/view.go`: render lyrics panel (synced highlight + "sin letra" state), artwork panel (with degradation), per-row cache indicator.
- [x] 3.8 Modify `main.go`: construct cache/lyrics/artwork/presence from config toggles, run cache `Sweep()` at startup, wire into `ui.New`; `defer presence.Close()`.

## Phase 4: Testing

- [x] 4.1 Create `internal/lyrics/lrc_test.go`: table tests for parse + `LineAt` boundaries/seek; plain-text fallback (lyrics: Synced/Plain, Seek updates highlight).
- [x] 4.2 Create `internal/lyrics/lyrics_test.go` (`httptest.Server`): synced found, no-match, API-down ⇒ no crash; DB cache hit skips HTTP (lyrics: No match / API down).
- [x] 4.3 Create `internal/cache/cache_test.go` (`t.TempDir()` + fake yt-dlp script): CRUD round-trip; `Lookup` invalidates missing/corrupt file; `Evict`/`Sweep` respect size+age budget + update index (download-cache: all scenarios).
- [x] 4.4 Create `internal/artwork/artwork_test.go`: `Detect` env-var matrix; unsupported ⇒ `None`/placeholder (artwork: Capable/Unsupported terminal).
- [x] 4.5 Create `internal/presence/presence_test.go`: empty `appID` and failing dialer ⇒ silent no-op (discord: Discord not running / Presence disabled).
- [x] 4.6 Modify `internal/storage/migrate_test.go`: assert migration 2 idempotent, `user_version` advances 1→2 (open twice on tmp DB).
- [x] 4.7 Create `internal/ui/update_test.go`: model-level tests cubren render de paneles letra/portada (incl. resaltado sincronizado, "sin letra", fallback plano), indicador de caché, fan-out en track-change (lookup→descarga condicional), descarte de respuestas obsoletas y paridad con la Fase 2 (toggles apagados ⇒ sin paneles ni Cmds extra). Se usó verificación a nivel de modelo en vez de `teatest`+golden para no introducir la dependencia `teatest` y mantener el binario puro Go; los frames golden de Bubble Tea quedan como deuda pendiente (ver nota abajo).

## Phase 5: Cleanup / Docs

- [x] 5.1 Update help text/README keybindings for lyrics/artwork panels + cache indicator + config toggle keys. (README: nuevo bloque de config con toggles cache/lyrics/artwork/presence, sección "Paneles de enriquecimiento" describiendo letra/portada/indicador `⤓`, y ruta de caché. Los paneles reaccionan automáticamente al cambio de pista; el tui-shell spec no define teclas nuevas, así que no se añadieron atajos.)
- [x] 5.2 Run `go vet ./...` + `go test ./...`; confirm `CGO_ENABLED=0` single static binary still builds (no cgo). (vet limpio; todos los paquetes en verde; `CGO_ENABLED=0 go build .` produce un ELF "statically linked" / `ldd` ⇒ "not a dynamic executable".)
