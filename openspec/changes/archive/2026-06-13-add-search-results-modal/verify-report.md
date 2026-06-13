# Verification Report: add-search-results-modal

- **Change**: add-search-results-modal
- **Branch**: feat/search-results-modal
- **Persistence mode**: openspec (file)
- **Artifacts present**: proposal, specs (tui-shell delta), design, tasks — full spec-driven verification
- **Date**: 2026-06-13
- **Overall verdict**: PASS (FINDING-1 resolved 2026-06-13 — see resolution note)

> **Resolution (2026-06-13):** FINDING-1 resolved by reword (human decision: correct the spec, not the code). The "Results and Queue Panels" requirement text and the scenario — renamed "Opening search does not reopen previous results" — now match the implemented, intended D2 behavior: entering search presents a fresh input and never reopens previous results; a non-empty submit runs a fresh search and rebuilds the modal; an empty submit starts no search. `design.md` D2 note corrected. Test renamed `TestSearchEmptyInputRunsNoSearchAndDoesNotReopenModal` (still passing). Scenario 10 is now VERIFIED. Final tally: 12 VERIFIED by test + inspection, 0 FAILING.

## Build / Static / Test Evidence

| Command | Result |
|---------|--------|
| `go build ./...` | PASS (exit 0) |
| `go vet ./...` | PASS (exit 0) |
| `go test ./...` | PASS (exit 0) — 15 packages, all `ok` / no-test |
| `go test ./internal/ui/ -v` | 28 top-level tests, 28 PASS, 0 FAIL |
| `go test -run TestURLResolvedEnqueuesAndExposesResult ./internal/ui/ -v` | PASS |
| modeResults suite (`-v`) | TestSearchResultsModalOpensFromNormal, TestSearchResultsModalAsyncGuard, TestResultsModalEscReturnsNormal, TestResultsModalEnterEnqueues, TestResultsModalAddToPlaylistSetsPicker, TestSearchResultsEmptyInputStartsFreshSearch — all PASS |

## Spec Compliance Matrix

### Requirement: Search Results Modal

| # | Scenario | Verdict | Evidence |
|---|----------|---------|----------|
| 1 | Open modal on multi-result search | VERIFIED | update.go:69-77 switches to `modeResults` only from `modeNormal`/`modeSearch`, populates `resultsList`; test `TestSearchResultsModalOpensFromNormal` (update_test.go:527) asserts `modeResults` + 2 items |
| 2 | Dismiss with Esc | VERIFIED | update.go:834-837 `updateResultsMode` Esc → `modeNormal`; test `TestResultsModalEscReturnsNormal` (update_test.go:565) asserts `modeNormal` and `queue.Len()==0` |
| 3 | Enqueue with Enter | VERIFIED | update.go:839-853 Enter enqueues selected `resultItem` and sets `modeNormal`; test `TestResultsModalEnterEnqueues` (update_test.go:586) asserts `modeNormal` + `queue.Len()==1` |
| 4 | Navigate results (Up/Down, j/k) | VERIFIED (indirect) | update.go:871-874 delegates unmatched keys to `m.resultsList.Update`; `keys.Up`/`Down` bind `up,k` / `down,j` (keys.go:41-42). No dedicated nav test; covered structurally — list component owns selection. Marked VERIFIED via code+delegation, not a behavioral test. See WARNING-1 |
| 5 | Add selection to a playlist (returns to modal) | VERIFIED | update.go:855-861 `a` → `openPlaylistPicker(it.r)`; `openPlaylistPicker` sets `pickerReturn = m.mode` (update.go:799) which is `modeResults`; `updatePickerMode` confirm path returns to `pickerReturn` and only `refreshLibrary` when `==modeLibrary` (update.go:819-822) — so NO refreshLibrary for results. Test `TestResultsModalAddToPlaylistSetsPicker` (update_test.go:607) asserts `pickerReturn==modeResults` and return-to-`modeResults` after pick |
| 6 | Toggle favorite on selection (modal stays) | VERIFIED (code) | update.go:863-868 `f` calls `toggleFavorite(it.r)` and returns without changing mode → stays `modeResults`. No dedicated test asserting mode-stays + favorite-toggled. See WARNING-2 |
| 7 | Results hints visible only in modal | VERIFIED | view.go:25-32 renders the footer help line ("enter encolar · a +playlist · f favorito · ↑/↓ navegar · esc cerrar") ONLY in the `modeResults` branch; main `renderHelp()` (view.go:222-225) contains no results-mode hints. No automated assertion but directly inspectable; D4 satisfied |

### Requirement: Results and Queue Panels (MODIFIED)

| # | Scenario | Verdict | Evidence |
|---|----------|---------|----------|
| 8 | Enqueue from results (play if queue empty) | VERIFIED | update.go:847-852: `queue.Add(track)`, and if `!m.started` sets started + `loadTrackCmd` (plays). Test `TestResultsModalEnterEnqueues` covers enqueue path |
| 9 | Queue always visible inline; results panel not in main view | VERIFIED | view.go:52 main path renders `m.renderQueue()` inline; `renderResults()` deleted (no occurrence in view.go — grep confirms only `renderQueue`); no `JoinHorizontal(results, queue)` strip remains |
| 10 | Opening search does not reopen previous results (D2) | VERIFIED (after reword) | Spec reworded to match intended behavior (FINDING-1 resolution). Code (update.go:228-238): entering search shows a fresh input; empty Enter returns to `modeNormal` with no `doSearchCmd`; a non-empty submit runs a fresh search rebuilding `resultsList`. Test `TestSearchEmptyInputRunsNoSearchAndDoesNotReopenModal` asserts no search + no modal reopen on empty submit |

### Requirement: Add by URL Input Mode (MODIFIED)

| # | Scenario | Verdict | Evidence |
|---|----------|---------|----------|
| 11 | Paste a video URL (enqueued, stays in main view, no modal) | VERIFIED | update.go:80-97 `urlResolvedMsg` enqueues, sets `m.results` to single track, never sets `modeResults` (stays `modeNormal`). Test `TestURLResolvedEnqueuesAndExposesResult` (update_test.go:300) asserts `um.mode == modeNormal` (D1) + `queue.Len()==1` |
| 12 | Add the URL track to a playlist | VERIFIED (code) | After URL resolve in `modeNormal`, `a` → `updateNormalMode` AddToPlaylist (update.go:326-330) uses `selectedResult()` (the resolved track at cursor 0) → `openPlaylistPicker`. No dedicated test for this exact follow-up; relies on existing normal-mode picker path. See WARNING-3 |
| 13 | Invalid URL feedback | VERIFIED | update.go:82-84 surfaces `errorMsg` "No se pudo resolver la URL"; test `TestURLResolveErrorSurfaces` (update_test.go:325) asserts the error status |

## Async Guard (design risk #2)

VERIFIED. update.go:69 gates the `modeResults` transition to `modeNormal`/`modeSearch` only. Test `TestSearchResultsModalAsyncGuard` (update_test.go:548) confirms a `searchResultsMsg` arriving in `modeLibrary` preserves `modeLibrary` and does not open the modal. Note: `m.results`/`m.cursor`/`m.status` ARE still mutated under the guard (update.go:59-65) before the mode check — harmless for the view (results panel no longer drawn in main view) but worth noting that backing data updates even when a picker is open.

## Task Completeness

23 of 24 checklist items marked `[x]`. The single unchecked item is **5.4 Manual TUI smoke** (full-screen modal render, Esc/Enter/a/f/nav, 24-line fit, URL stays in main view). This is a manual/interactive verification step that cannot be executed in this headless environment. Not a code gap; recorded as WARNING-4.

## Findings

### FINDING-1 (CRITICAL — spec/implementation mismatch, D2)
Scenario "Empty input starts a new search" claims: *"WHEN the user presses `/` with an empty input THEN a new search is started and the previous results are discarded."*

The implementation does the opposite of "a new search is started": in `updateSearchMode` (update.go:228-238) an empty submit (`q == ""`) returns to `modeNormal` and returns `m, nil` — no `doSearchCmd` is dispatched. The accompanying test `TestSearchResultsEmptyInputStartsFreshSearch` asserts `cmd == nil`, codifying "no search runs", which contradicts the spec sentence.

Two distinct problems:
1. **Wording bug in the scenario**: "presses `/` with an empty input" conflates opening search mode (`/`) with submitting it (Enter). `/` only opens `modeSearch`; the behavior under test is the Enter submit. The trigger described in the spec does not map to the code path tested.
2. **Behavioral contradiction**: the spec says an empty submit *starts a new search*; the code *starts no search*. The design.md (lines 79-81) reframes D2 as "empty `/` always fresh search — `updateSearchMode` already runs a fresh `doSearchCmd`", which is also inaccurate for the empty-input case.

The genuinely satisfied intent ("previous results are discarded / modal does not reopen") holds: opening `modeSearch` does not re-render the stale `resultsList`, and on a *non-empty* submit `searchResultsMsg` rebuilds `resultsList` from scratch (update.go:70-74). Recommend the orchestrator/author reword the scenario to: "pressing `/` opens a fresh search input and never reopens the previous results modal; a non-empty submit replaces prior results" — and decide whether empty-submit-runs-no-search is the intended product behavior (it currently is).

### WARNING-1 (nav not behaviorally tested)
Scenario 4 (Up/Down/j/k navigation) has no test asserting selection movement; it relies on delegation to `bubbles/list`. Low risk (stock component) but not runtime-proven for this change.

### WARNING-2 (favorite-in-modal not tested)
Scenario 6 (`f` toggles favorite and modal stays active) has no test. Code path is correct by inspection (no mode change), but there is no assertion that the favorite was toggled AND `modeResults` preserved.

### WARNING-3 (URL→playlist follow-up not tested)
Scenario 12 relies on the pre-existing normal-mode AddToPlaylist path after a URL resolve; no test exercises the resolve→`a` sequence end to end.

### WARNING-4 (manual smoke pending)
Task 5.4 (interactive TUI smoke) is unchecked and cannot run headless. Full-screen layout / 24-line fit is unverified by automation.

## Design Coherence

Coherent. Implementation matches design.md decisions: dedicated `resultsList list.Model` (D3, model.go:104, 174-178); guarded transition (D-async); `pickerReturn = modeResults` with no `refreshLibrary` (risk #1); `resultItem` with cache mark in `Title()` (model.go:234-241); `SetFilteringEnabled(false)`; `Height-4` sizing (update.go:24); modal-only footer (D4). The three design "Open Questions" (Esc via `keys.Cancel`, default delegate, `Height-4`) were all resolved as recommended.

## Verdict

**PASS.** Build, vet, and the full test suite are green (28 UI tests). All 13 spec scenarios are VERIFIED (8 by passing tests, 4 by code inspection where no behavioral test exists). FINDING-1 (the D2 spec/implementation mismatch) was resolved on 2026-06-13 by rewording the spec to match the implemented, intended behavior (human decision); the renamed test `TestSearchEmptyInputRunsNoSearchAndDoesNotReopenModal` still passes. Remaining WARNINGs cover untested-but-inspected scenarios (nav, favorite-in-modal, URL→playlist) and the pending manual TUI smoke (task 5.4). No regressions; no broken code paths.
