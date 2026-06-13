# Archive Report: add-url-import-and-lyrics-memory

**Archived**: 2026-06-13
**Artifact store**: openspec
**Final phase**: archive (completed)

## Summary

Change covering YouTube URL ingestion (single video + playlist import), manual lyrics
search with persisted lyrics reference (remembered query / provider id reused on replay),
and a clear-queue shortcut. The PR for this change was already merged to `master` and is
in production at archive time.

## Override Notice

**EXPLICIT USER OVERRIDE — "solo archivar".** The formal `verify` and `judge` phases were
skipped because the implementation is already merged and running in production. This
override authorized proceeding past the normal verify/judge gate. The skipped phases are
recorded in `state.yaml` history as `verify: skipped-override` and `judge: skipped-override`
(ts 2026-06-13).

No `verify-report.md` exists for this change; per the strict-vs-OpenSpec policy this would
normally block, but the user's intentional-archive override covers it. This is an
intentional-with-warnings archive.

## Specs Synced (deltas merged into openspec/specs/)

| Domain | Action | Details |
|--------|--------|---------|
| lyrics | MODIFIED + ADDED | Modified "Fetch Lyrics" (saved-reference reuse before auto query); added "Manual Lyrics Search" and "Persist Lyrics Reference" |
| playback-queue | ADDED | "Clear Queue" |
| playlists | ADDED | "Import Playlist from YouTube URL" |
| tui-shell | ADDED | "Add by URL Input Mode", "Import Playlist Mode", "Manual Lyrics Search Mode", "Clear Queue Shortcut" |
| youtube-search | ADDED | "Resolve a Video URL", "Resolve a Playlist URL" |

All pre-existing requirements in each main spec were preserved. No REMOVED or RENAMED
deltas were present, so no destructive merges occurred.

## Task Completion

All implementation tasks (Phases 1-4) are checked complete. `go build ./...` and
`go test ./...` verification checkboxes are checked.

## Deferred Debt

- **Manual TUI smoke test** (`tasks.md` Verification, last item) remains unchecked:
  "Smoke manual de TUI: URL→cola(+playlist), import→playlist, letra manual→persiste/reusa,
  limpiar→vacía+para." This is a manual, non-implementation verification step. Since the
  change is merged and in production, it is recorded here as deferred debt rather than
  blocking the archive. Recommend a manual smoke pass against the production build to close
  it out.
- No `verify-report.md` / formal judge pass exists (see Override Notice).
