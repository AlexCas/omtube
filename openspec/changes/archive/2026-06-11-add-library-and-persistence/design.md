# Design: Library and Persistence (Fase 2)

## Technical Approach

A SQLite-backed `internal/storage` package (pure-Go `modernc.org/sqlite`, not yet in
go.mod) opens `config.LibraryFile()`; a versioned migration runner builds the schema on
startup. Per-entity repositories (tracks, playlists, favorites, history) expose CRUD
returning `error`, never panicking. Domain packages `internal/playlist` and
`internal/favorites` wrap repos with validation. `internal/history` is reimplemented on
the history repo, keeping its `Entry`/`Add`/`Entries` shape so `main.go`/`ui` barely
change. The TUI gains a `modeLibrary` reusing `queue.Add` to play playlists. Tracks are
keyed by `search.Result.ID` (video id) per the track-identity spec. Wiring stays in
`main.go`/`ui.New`.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Driver | `modernc.org/sqlite` (pure Go) via `database/sql` | `mattn/go-sqlite3` (cgo) | Keeps single static binary; cgo breaks cross-build |
| Migrations | Ordered `[]string` + `PRAGMA user_version`; apply `>current` in a tx | golang-migrate / goose | Spec needs only simple/ordered/idempotent; no heavy dep |
| Repo layout | Shared `*sql.DB`; repo structs hold it; thin methods + raw SQL | ORM (gorm/ent) | Tiny schema; reviewable, dep-free, matches hand-rolled style |
| Track upsert | `INSERT ... ON CONFLICT(video_id) DO UPDATE` before referencing | dedupe in Go | DB-enforced identity reuse (track-identity spec) |
| `history` API | Keep `*History` `Add`/`Entries`; back with repo | New API | Minimizes `main.go`/`ui` churn; satisfies MODIFIED persist-to-DB |
| History order | Repo recent-first for Browse; `Entries()` stays oldest-first | one order | Browse spec wants recent-first; Fase 1 test expects oldest-first |
| Library UI | `modeLibrary`, sections playlists/favorites/history via `bubbles/list` | always-on panel | Isolated mode leaves normal view intact (tui-shell reqs) |
| Keybindings | `L` library, `f` favorite, `a` add-to-playlist, `esc` back | reuse enter | Avoids space/`n`/`p`/`+`/`-`/`/`/`q` collisions |

## Data Flow

    ui.Update ─▶ favorites.Toggle / playlist.Add ─▶ repo ─▶ SQLite (library.db)
        │                                                      ▲
        ├─▶ playlist.Play ─▶ []search.Result ─▶ queue.Add ─▶ player.Load
        │
    loadedMsg ─▶ history.Add ─▶ historyRepo.Insert ─▶ SQLite
    startup ─▶ storage.Open ─▶ migrate ─▶ importLegacyJSON(history.json → DB, keep backup)

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod`/`go.sum` | Modify | Add `modernc.org/sqlite` (+indirect) |
| `internal/storage/storage.go` | Create | `Open`, `*DB`, `Close`, pragmas (WAL, foreign_keys) |
| `internal/storage/migrate.go` | Create | Ordered migrations + `user_version` runner |
| `internal/storage/tracks.go` | Create | `TrackRepo` upsert/get |
| `internal/storage/playlists.go` | Create | `PlaylistRepo` CRUD + `playlist_tracks` |
| `internal/storage/favorites.go` | Create | `FavoriteRepo` toggle/list/exists |
| `internal/storage/history.go` | Create | `HistoryRepo` insert/list recent-first |
| `internal/storage/*_test.go` | Create | CRUD round-trips on tmp DB |
| `internal/playlist/playlist.go` | Create | Validation, dup/rename rules, `Play` |
| `internal/favorites/favorites.go` | Create | Idempotent toggle, list |
| `internal/history/history.go` | Modify | Back with `HistoryRepo`; `Browse()`; JSON import |
| `internal/history/history_test.go` | Modify | Storage-backed |
| `internal/config/config.go` | Modify | `LibraryFile()` → `DataDir/library.db` |
| `main.go` | Modify | `storage.Open`, build repos, `defer db.Close()` |
| `internal/ui/model.go` | Modify | Repos/services + `modeLibrary` state |
| `internal/ui/keys.go` | Modify | Library/Favorite/AddToPlaylist bindings |
| `internal/ui/update.go` | Modify | `updateLibraryMode`; fav/add/play handlers |
| `internal/ui/view.go` | Modify | `renderLibrary`; help text |

## Interfaces / Contracts

```go
func Open(path string) (*DB, error) // opens, migrates, sets pragmas
type TrackRepo    struct{ db *sql.DB } // Upsert(search.Result); Get(id)→(Result,bool,error)
type PlaylistRepo struct{ db *sql.DB } // Create/Rename/Delete; Add/Remove(plID,videoID); Tracks(plID); List()
type FavoriteRepo struct{ db *sql.DB } // Add/Remove(videoID); Exists(videoID); List()
type HistoryRepo  struct{ db *sql.DB } // Insert(Entry); List(limit) recent-first

func playlist.New(*PlaylistRepo, *TrackRepo) *Service  // ErrEmptyName/ErrDuplicateName/ErrNotFound
func favorites.New(*FavoriteRepo, *TrackRepo) *Service // Toggle(search.Result)→(bool,error)
```

Schema (migration 1): `tracks(video_id PK, title, uploader, duration)`;
`playlists(id PK, name UNIQUE, created_at)`;
`playlist_tracks(playlist_id, video_id, position, PK(playlist_id,video_id))` FK→both;
`favorites(video_id PK, created_at)` FK; `history(id PK, video_id, played_at)` FK.
`PRAGMA foreign_keys=ON`.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|--------------|----------|
| Unit | Migrations idempotent; `user_version` advances once | open twice on tmp DB |
| Unit | Repo CRUD round-trips; missing read returns empty+nil err | tmp file DB per `t.TempDir()` |
| Unit | Playlist dup/empty/rename-collision; favorite idempotency | table tests on real repo |
| Integration | Legacy `history.json` import on first run + backup preserved | seed JSON, open, assert rows + file kept |
| E2E | Library mode open/nav, toggle fav, add-to-playlist, play playlist | manual TUI smoke |

## Migration / Rollout

`storage.Open` advances `user_version` to latest. `history.Load` does a one-time legacy
import: if the `history` table is empty AND `history.json` exists, parse it, bulk-insert
entries, then rename `history.json` → `history.json.bak` (never delete) so re-runs are
no-ops. Absent JSON = empty history, no error. Rollback = run the Fase 1 binary;
`history.json[.bak]` is untouched and `library.db` is ignored by it.

## Open Questions

- [ ] `LibraryFile()` config-override key vs. fixed XDG path (default: fixed)
- [ ] Add-to-playlist UI: inline picker vs. text prompt (lean: `bubbles/list` picker)
