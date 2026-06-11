# Tasks: Library and Persistence (Fase 2)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~700-900 (11 new + 7 modified + migrations + tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (backend) → PR 2 (integration/UI) |
| Delivery strategy | exception-ok |
| Chain strategy | feature-branch-chain (2 PRs) |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units (2-PR split — user decision)

| Unit | Goal | PR | Tasks | Notes |
|------|------|----|-------|-------|
| 1 | Persistence backend: storage layer + repos + playlist/favorites domain + unit tests | PR 1 | 1.1–1.7, 2.1, 2.2, 4.1–4.4 | Standalone. Only existing-file change is `go.mod`. Compiles + tests green without touching history/main/UI. ~350–450 lines. |
| 2 | Integration & UI: history-on-repo + JSON import, config path, main wiring, library mode, integration tests, docs | PR 2 | 2.3, 3.1–3.6, 4.5, 5.1, 5.2 | Depends on PR 1. Wires backend into the app. ~350–450 lines. |

Each PR builds and tests independently. Applied sequentially (PR 1 fully green before PR 2). No git repo present, so PRs materialize as two ordered apply slices.

## Phase 1: Foundation / Storage

- [x] 1.1 Add `modernc.org/sqlite` to `go.mod`/`go.sum` (`go get`, `go mod tidy`).
- [x] 1.2 Create `internal/storage/storage.go`: `Open(path)→(*DB,error)`, `Close`, set `WAL` + `foreign_keys=ON` pragmas.
- [x] 1.3 Create `internal/storage/migrate.go`: ordered `[]string` + `PRAGMA user_version` runner applying only `>current` in a tx (migration 1 = full schema).
- [x] 1.4 Create `internal/storage/tracks.go`: `TrackRepo.Upsert(search.Result)` ON CONFLICT(video_id), `Get(id)→(Result,bool,error)`.
- [x] 1.5 Create `internal/storage/playlists.go`: `PlaylistRepo` Create/Rename/Delete/List + Add/Remove/Tracks over `playlist_tracks` (position-ordered).
- [x] 1.6 Create `internal/storage/favorites.go`: `FavoriteRepo` Add/Remove/Exists/List.
- [x] 1.7 Create `internal/storage/history.go`: `HistoryRepo` Insert/List(limit) recent-first.

## Phase 2: Domain Packages

- [x] 2.1 Create `internal/playlist/playlist.go`: `New(*PlaylistRepo,*TrackRepo)`; ErrEmptyName/ErrDuplicateName/ErrNotFound; dup-track guard; `Play()→[]search.Result`.
- [x] 2.2 Create `internal/favorites/favorites.go`: `New(*FavoriteRepo,*TrackRepo)`; idempotent `Toggle(Result)→(bool,error)`, `List`.
- [x] 2.3 Modify `internal/history/history.go`: back `Entry`/`Add`/`Entries` with `HistoryRepo`; add `Browse()` recent-first; one-time `history.json`→DB import keeping `.bak`.

## Phase 3: Integration / UI

- [x] 3.1 Modify `internal/config/config.go`: add `LibraryFile()` → `DataDir/library.db` (fixed XDG, no override).
- [x] 3.2 Modify `main.go`: `storage.Open`, build repos+services, `defer db.Close()`, pass to `ui.New`.
- [x] 3.3 Modify `internal/ui/model.go`: hold repos/services + `modeLibrary` state.
- [x] 3.4 Modify `internal/ui/keys.go`: bind `L`/`f`/`a`/`esc` (no collision with space/`n`/`p`/`+`/`-`/`/`/`q`).
- [x] 3.5 Modify `internal/ui/update.go`: `updateLibraryMode`; favorite-toggle, add-to-playlist (`bubbles/list` picker), play-playlist via `queue.Add`.
- [x] 3.6 Modify `internal/ui/view.go`: `renderLibrary` (playlists/favorites/history sections) + help text.
- [x] 3.7 Add create-playlist UI flow: bind `c` in library mode → text-input prompt → `playlist.Service.Create`; reject empty/duplicate with a message (tui-shell: Create Playlist from UI).

## Phase 4: Testing

- [x] 4.1 Create `internal/storage/migrate_test.go`: open twice on tmp DB; `user_version` advances once, idempotent (lib-persistence: Already up to date).
- [x] 4.2 Create `internal/storage/*_test.go`: CRUD round-trips per repo; missing read returns empty+nil err (lib-persistence: CRUD round-trip / Read missing).
- [x] 4.3 Add playlist tests: empty/dup name, rename collision, dup-track no-op, empty-playlist play (playlists spec).
- [x] 4.4 Add favorites tests: mark idempotent, unmark non-favorite no-op (favorites spec).
- [x] 4.5 Modify `internal/history/history_test.go`: storage-backed; legacy JSON import + `.bak` preserved (playback-history: Import existing JSON).

## Phase 5: Cleanup / Docs

- [x] 5.1 Update help text/README keybinding list (`L`/`f`/`a`) for library mode.
- [x] 5.2 Run `go vet ./...` + `go test ./...`; confirm single static binary (no cgo).
