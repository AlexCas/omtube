# Design: Media Enrichment (Fase 3)

## Technical Approach

Four new pure-Go packages: `internal/cache` (yt-dlp audio → XDG cache + SQLite index in the
existing `library.db`), `internal/lyrics` (stdlib `net/http` lrclib client + `.lrc` parser),
`internal/artwork` (terminal graphics detect + render), `internal/presence` (Discord IPC).
The player gains an `EventTrackChange` and a cache-aware `Load`: `ui` resolves the cached
path (else the YouTube URL) before `Load`. Subscribers stay decoupled via the *existing*
Bubble Tea fan-out — a track-change event spawns `tea.Cmd`s that fetch lyrics/artwork and set
presence; no goroutine bus, no inter-package calls. Each feature is a toggle and degrades to
a no-op on failure, so the core loop never breaks.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Subscriber model | Player emits `EventTrackChange`; `ui.Update` dispatches `tea.Cmd`s to each feature | shared goroutine event bus; observers in player | Reuses existing `waitForEventCmd` fan-out; keeps player ignorant of lyrics/art/presence (no coupling) |
| Cache index | New tables in existing `library.db` via migration 2 | separate `cache.db`; JSON sidecar | One DB, one migration runner, FK to `tracks`; atomic with library |
| Cache files | yt-dlp `-x` (extract-audio) to `~/.cache/terminaltube/audio/<id>.<ext>` | mpv `--stream-record` | Decoupled from playback; reusable, validatable, evictable |
| Eviction | Size+age budget; delete oldest by `last_used` until under limit, then update index | LRU in memory; no limit | Survives restarts; spec wants size/age caps |
| Lyrics client | stdlib `net/http` to lrclib (no auth); cache rows in DB | add HTTP dep; scrape | Zero new deps; pure Go; lrclib serves synced `.lrc` |
| `.lrc` parse + sync | In-package parser → `[]Line{T, Text}`; UI binary-searches by `pos` | external lrc lib; mpv subs | Tiny, testable, drives highlight from existing `posMsg` tick |
| Artwork detect | Probe `$TERM`/`$KITTY_WINDOW_ID`/`$TERM_PROGRAM` + env; order kitty→sixel→chafa→none | terminal DA1 query | No raw-mode round-trip in TUI; degrades cleanly |
| Discord lib | `github.com/hugolgst/rich-go` (pure Go Unix-socket IPC) | hand-rolled IPC; disgord | Pure Go (no cgo), tiny, connect-failure = silent no-op |
| Failure policy | Toggle off OR error ⇒ feature no-ops; log at warn | surface errors in UI | Specs require silent degradation; core playback unaffected |

## Data Flow

    ui.Enqueue/Next ─▶ cache.Lookup(id) ─▶ hit: localPath / miss: result.URL()
                                              │
                                              ▼
                                       player.Load(src)
                                              │ emits EventTrackChange{Result, src}
                                              ▼
              playerEventMsg ─▶ ui.Update fans out tea.Cmds:
                 ├─▶ lyrics.Fetch(title,artist) ─▶ lyricsMsg ─▶ panel
                 ├─▶ artwork.Render(thumbURL)   ─▶ artworkMsg ─▶ panel
                 ├─▶ presence.Set(title,artist) (fire-and-forget)
                 └─▶ cache.Download(id) [if miss & enabled] ─▶ index row + evict
    tickMsg/posMsg ─▶ lyrics.LineAt(pos) ─▶ highlight advances

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod`/`go.sum` | Modify | Add `github.com/hugolgst/rich-go` (pure Go) |
| `internal/cache/cache.go` | Create | `Service`: `Lookup`, `Download` (yt-dlp `-x`), `Validate`, `Evict`, `Clear` |
| `internal/cache/index.go` | Create | `cache_entries` CRUD on shared `*sql.DB` |
| `internal/storage/migrate.go` | Modify | Migration 2: `cache_entries`, `lyrics_cache` |
| `internal/storage/storage.go` | Modify | `Cache()`/`Lyrics()` repo accessors |
| `internal/lyrics/lyrics.go` | Create | lrclib HTTP client + DB-cached fetch |
| `internal/lyrics/lrc.go` | Create | `.lrc` parser; `Lyrics.LineAt(sec)` |
| `internal/artwork/artwork.go` | Create | `Detect()`, `Render(url,w,h)`; kitty/sixel/chafa/none |
| `internal/presence/presence.go` | Create | `Connect`, `Set`, `Clear`, `Close`; silent on failure |
| `internal/player/player.go` | Modify | Add `EventTrackChange`; `Event` carries `Track`/`Source` |
| `internal/player/mpv.go` | Modify | `Load(src)` loads local file or URL; emit track-change |
| `internal/config/config.go` | Modify | Toggles + cache dir/limits; `CacheDir()` |
| `internal/ui/messages.go` | Modify | `lyricsMsg`/`artworkMsg`; fetch/render/presence Cmds |
| `internal/ui/model.go` | Modify | Hold cache/lyrics/artwork/presence; panel + cache-flag state |
| `internal/ui/update.go` | Modify | Dispatch on track-change; cache lookup before Load; highlight on tick |
| `internal/ui/view.go` | Modify | Lyrics + artwork panels; per-row cache indicator |
| `main.go` | Modify | Construct services from toggles; wire into `ui.New` |
| `*_test.go` (cache/lyrics/artwork) | Create | Unit tests per below |

## Interfaces / Contracts

```go
// player
const EventTrackChange EventKind = iota + 2
type Event struct { Kind EventKind; Track search.Result; Source string }
Load(src string) error // local file path or YouTube URL

cache.New(repo, ytdlp, dir string, maxBytes int64, maxAge time.Duration) *Service
  Lookup(id string) (path string, ok bool) // validates file exists
  Download(ctx, r search.Result) (path string, err error); Evict() error; Clear() error
lyrics.New(repo, httpClient) *Service
  Fetch(ctx, title, artist string, dur int) (Lyrics, error) // Lyrics{Synced bool; Lines []Line; Plain string}
  (Lyrics) LineAt(sec float64) int
artwork.Detect() Backend // Kitty|Sixel|Chafa|None
  (Backend) Render(ctx, thumbURL string, w, h int) (string, error) // escape seq or placeholder
presence.New(appID string) *Client // Connect() error (silent); Set(title,artist string); Clear(); Close()
```

Migration 2: `cache_entries(video_id PK, path, size_bytes, ext, created_at, last_used)`
FK→`tracks` ON DELETE CASCADE; `lyrics_cache(video_id PK, synced INT, body TEXT, fetched_at)`
FK→`tracks`. Config keys: `cache.enabled`, `cache.max_size_mb`, `cache.max_age_days`,
`lyrics.enabled`, `artwork.enabled`, `presence.enabled`, `presence.app_id`.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `.lrc` parse + `LineAt` boundaries/seek; plain fallback | table tests, no network |
| Unit | lyrics fetch success/no-match/down ⇒ no-crash; DB cache hit skips HTTP | `httptest.Server` |
| Unit | cache index CRUD; `Lookup` invalidates missing/corrupt file; eviction respects budget | `t.TempDir()` + fake yt-dlp script |
| Unit | artwork `Detect` per env matrix; unsupported ⇒ `None`/placeholder | env-var table tests |
| Unit | presence connect-failure is silent no-op | inject failing dialer |
| Integration | migration 2 idempotent; advances `user_version` 1→2 | open twice on tmp DB |
| E2E | toggles off ⇒ app behaves as Fase 2; panels render; cache replay skips download | manual TUI smoke |

**UI test debt (Fase 2, 0% coverage):** this design makes `ui` testable — lyrics/artwork
logic lives in pure functions (`LineAt`, `Detect`, `Render`), unit-testable outside Bubble
Tea. Full `Model.Update` coverage via `teatest` (golden panel frames) is recommended in
tasks; the high-value first slice is unit-testing those extracted helpers.

## Migration / Rollout

Migration 2 only ADDS tables — Fase 2 data untouched; downgrade = run Fase 2 binary (extra
tables ignored). Defaults on missing config: cache on, lyrics on, artwork on, presence off
(needs `app_id`). Rollback = toggles false + `rm -rf ~/.cache/terminaltube/`; `library.db`
keeps working.

## Resolved Decisions (human review gate, 2026-06-12)

- [x] **Discord `presence.app_id`**: user-supplied. Presence stays disabled until `presence.app_id` is set in config; no default TerminalTube application id is shipped. Toggling `presence.enabled=true` without an `app_id` is a silent no-op (logged once).
- [x] **Cache eviction trigger**: post-download **and** startup sweep. Eviction runs after each successful cache write and once at app startup, so space is reclaimed across sessions and after the user lowers the size/age limit.
- [x] **Artwork source**: reuse the cache download when caching (yt-dlp `--write-thumbnail` alongside the audio), otherwise fetch the YouTube thumbnail URL on track-change. No separate thumbnail fetch when a cached thumbnail already exists.
