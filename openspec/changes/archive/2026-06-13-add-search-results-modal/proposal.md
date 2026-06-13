# Proposal: Move Search Results into a Full-Screen Takeover Modal

## Intent

The always-visible main `View()` (`internal/ui/view.go`) stacks title, search input,
a side-by-side Results+Queue strip (`JoinHorizontal`), lyrics, cover, now-playing,
help, and the animated visualizer. On a standard 24-line terminal this overflows on
some OSes — the widest contributor is the results+queue horizontal strip, and the
results panel is rendered even while just listening (showing "(vacío)").

Move the search-RESULTS panel out of the main view into a full-screen takeover modal
(`modeResults`), reusing the established `m.picker.View()` pattern (`view.go:20-22`)
backed by `bubbles/list`. The main view keeps: search input, queue (now inline,
full-size), lyrics, cover, instructions, animated visualizer.

## Scope

### In Scope
- New `modeResults` takeover modal backed by a **dedicated** `resultsList list.Model` (D3).
- Main-view reflow: remove the `JoinHorizontal(results, queue)` strip; promote queue
  to an inline full-size panel; drop `renderResults()` from the main `View()` path.
- Key routing: `searchResultsMsg` (multi-result) transitions to `modeResults`;
  `updateResultsMode` handles Esc (dismiss), Enter (enqueue + return), Up/Down/j/k
  (navigate), `a` (add-to-playlist), `f` (favorite). `WindowSizeMsg` sizes `resultsList`.
- D1: A single track resolved from a URL does NOT open the modal — stays in main view.
- D2: Pressing `/` with empty input ALWAYS starts a new search; prior results discarded.
- D4: Results-mode key hints appear ONLY while the modal is active.

### Out of Scope
- Floating/overlay compositing (Approach C, rejected — no Lipgloss primitive).
- Backend search, URL resolution, or playback changes.
- Changes to the URL single-track flow (D1).
- Persistent "reopen last results" (D2).

## Capabilities

> Contract for sdd-spec. Researched against `openspec/specs/`.

### New Capabilities
- None.

### Modified Capabilities
- `tui-shell`: "Results and Queue Panels" requirement changes — results now display
  in a full-screen `modeResults` modal (entered after a multi-result search) instead of
  an always-visible side-by-side panel; queue becomes an inline full-size panel. Add
  modal lifecycle scenarios (open on multi-result search, dismiss with Esc, enqueue with
  Enter, `a`/`f` on selection) and the D1/D2/D4 behaviors. "Add by URL Input Mode"
  confirms the single resolved track stays in the main view (D1).

## Approach

Approach A from exploration (recommended). Add a `modeResults` branch at the top of
`View()` returning `m.resultsList.View()`, mirroring `modePicker`/`modeLyricsPicker`.
Wrap `search.Result` in a small `resultItem` adapter (like `playlistItem`/`candidateItem`),
embedding the cache indicator in `Title()`. Disable list filtering
(`SetFilteringEnabled(false)`) so `/` is not swallowed. Guard the
`searchResultsMsg` auto-transition to `modeNormal`/`modeSearch` only. `updateResultsMode`
mirrors `updatePickerMode`; `pickerReturn = modeResults` must restore correctly on `a`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/ui/model.go` | Modified | Add `modeResults` const + `resultsList list.Model` field + `resultItem` adapter |
| `internal/ui/view.go` | Modified | Add `modeResults` takeover branch; remove `renderResults()` + `JoinHorizontal`; inline queue |
| `internal/ui/update.go` | Modified | `searchResultsMsg` → `modeResults` (guarded); add `updateResultsMode`; size `resultsList` on `WindowSizeMsg` |
| `internal/ui/keys.go` | Reviewed | Existing `Cancel`/`Enqueue`/`Up`/`Down`/`Favorite`/`AddToPlaylist` cover it; no new bindings |
| `internal/ui/update_test.go` | Modified | Fix `TestURLResolvedEnqueuesAndExposesResult` (mode stays normal per D1); add `modeResults` transition tests |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `pickerReturn` returns to wrong mode after `a` | Med | Verify `updatePickerMode` restores `modeResults` without `refreshLibrary` |
| Async results hijack a non-normal view | Med | Guard transition to `modeNormal`/`modeSearch` only |
| `resultsList` renders zero-size until resize | Low | Add `SetSize` in `WindowSizeMsg`; tests set width/height |
| `bubbles/list` `/` key conflict | Low | `SetFilteringEnabled(false)` (same as `m.picker`) |

## Rollback Plan

Single-PR, `internal/ui`-scoped. Revert the PR (or the commit) to restore the
always-visible side-by-side results panel; no data, schema, or backend migration.

## Dependencies

- None new. `bubbles/list` already in use.

## Success Criteria

- [ ] Main view fits a standard 24-line terminal (no results strip).
- [ ] A multi-result search opens `modeResults`; Esc/Enter/`a`/`f`/nav work.
- [ ] URL single-track resolve stays in main view (D1); empty `/` starts fresh (D2).
- [ ] Results-mode hints shown only in `modeResults` (D4); dedicated `resultsList` (D3).
- [ ] `go test ./...` green, including updated `TestURLResolvedEnqueuesAndExposesResult`.
