# Tasks: Move Search Results into a Full-Screen Takeover Modal

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~120-160 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception (not needed; under budget) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Full modeResults modal + tests | PR 1 | Single, internal/ui-scoped; build green at each phase |

## Phase 1: Model Foundation (build stays green)

- [x] 1.1 In `internal/ui/model.go`, append `modeResults` to the mode iota (after `modeLyricsPicker`).
- [x] 1.2 In `internal/ui/model.go`, add `resultItem` adapter: `Title()` = `mark + r.Title`, `Description()` = `r.Uploader`, `FilterValue()` = `r.Title`.
- [x] 1.3 In `internal/ui/model.go`, add `resultsList list.Model` field to the model.
- [x] 1.4 In `New()`, build `resultsList` with `list.NewDefaultDelegate()`, Title "Resultados", `SetShowHelp(false)`, `SetShowStatusBar(false)`, `SetFilteringEnabled(false)`.

## Phase 2: View Branch (build stays green)

- [x] 2.1 In `internal/ui/view.go`, add a `modeResults` branch at the top of `View()` returning `m.resultsList.View()` plus the modal-only footer help line ("enter encolar · a +playlist · f favorito · ↑/↓ navegar · esc cerrar") (D4).
- [x] 2.2 In `internal/ui/view.go`, drop the `JoinHorizontal(results, queue)` strip from the main path and render `renderQueue()` inline at full size.
- [x] 2.3 In `internal/ui/view.go`, delete `renderResults()`; confirm `renderHelp()` carries no results-mode hints (D4).

## Phase 3: Update Routing (build stays green)

- [x] 3.1 In `internal/ui/update.go` `WindowSizeMsg`, add `m.resultsList.SetSize(msg.Width, msg.Height-4)` beside the picker sizing.
- [x] 3.2 In `searchResultsMsg` handler, build `[]list.Item` of `resultItem` (mark via `m.cacheMark(r.ID)`), `SetItems`, `Select(0)`, and switch to `modeResults` ONLY when `m.mode == modeNormal || m.mode == modeSearch` (guard).
- [x] 3.3 Confirm `urlResolvedMsg` stays in `modeNormal` and does NOT open the modal (D1).
- [x] 3.4 Add `updateResultsMode` mirroring `updatePickerMode`: Esc (`keys.Cancel`) → `modeNormal`; Enter → enqueue selected (reuse normal-mode enqueue/`started`/`loadTrackCmd`) + `modeNormal`; `a` → `openPlaylistPicker(selected)` setting `pickerReturn = modeResults`; `f` → `toggleFavorite(selected)` stay; default → `m.resultsList.Update`.
- [x] 3.5 Route `modeResults` KeyMsg to `updateResultsMode` in the main `Update` switch.

## Phase 4: Testing

- [x] 4.1 Update `TestURLResolvedEnqueuesAndExposesResult` to assert `um.mode == modeNormal` after URL resolve (D1).
- [x] 4.2 Add test: `searchResultsMsg` with 2+ results from `modeNormal` → `modeResults`, items populated.
- [x] 4.3 Add test: same msg from `modeLibrary` → stays `modeLibrary`, no modal (async guard).
- [x] 4.4 Add test: `modeResults` + Esc → `modeNormal`, queue unchanged.
- [x] 4.5 Add test: `modeResults` + Enter → selected track in queue, `modeNormal`.
- [x] 4.6 Add test: `modeResults` + `a` → `modePicker` with `pickerReturn == modeResults`; pick → back to `modeResults`, no `refreshLibrary`.
- [x] 4.7 Add test: empty `/` from prior results starts a fresh search, modal does not reopen (D2).
- [x] 4.8 Use `newTestModel` (width=120, height=40) so `resultsList` has non-zero size.

## Phase 5: Verification

- [x] 5.1 `go build ./...` succeeds.
- [x] 5.2 `go vet ./...` clean.
- [x] 5.3 `go test ./...` green (including updated and new modal tests).
- [ ] 5.4 Manual TUI smoke: run a multi-result search → modal opens full-screen; Esc/Enter/`a`/`f`/nav work; main view fits 24 lines; URL resolve stays in main view.
