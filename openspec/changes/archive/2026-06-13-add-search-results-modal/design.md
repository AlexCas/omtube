# Design: Move Search Results into a Full-Screen Takeover Modal

## Technical Approach

Approach A from exploration. Add a `modeResults` takeover backed by a dedicated
`resultsList list.Model`, mirroring the `modePicker`/`modeLyricsPicker` precedent
(`view.go:20-22`, `update.go:790-813`). A multi-result `searchResultsMsg`
populates `resultsList` and switches to `modeResults` (guarded); `View()` returns
`m.resultsList.View()` for that mode. The main view drops the
`JoinHorizontal(results, queue)` strip and keeps the queue inline. Satisfies the
`tui-shell` delta (Search Results Modal, Results and Queue Panels, Add by URL).

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Results state | Dedicated `resultsList list.Model` (D3); keep `m.results`/`m.cursor` as backing data | Cleaner than a 3rd reuse of `m.picker`; `m.results`/`cursor` still feed `selectedResult()` for D1 normal-mode actions |
| `searchResultsMsg` transition | Switch to `modeResults` only when `m.mode == modeNormal \|\| m.mode == modeSearch` | Async results must not hijack `modeLibrary`/pickers (risk #2) |
| `urlResolvedMsg` (D1) | Stay in `modeNormal`; do NOT open modal | Single auto-enqueued track; modal for one item is redundant |
| `a` from results | `pickerReturn = modeResults`; do NOT `refreshLibrary` | Returns to modal, not library (risk #1) |
| List filtering | `SetFilteringEnabled(false)` | Prevents `/` being swallowed (same as `m.picker`) |
| Cache indicator | Folded into `resultItem.Title()` | `list.Item` has no `cacheMark` hook; matches `candidateItem` prefix pattern |
| Results help line | Custom footer string in modal only (D4) | List `SetShowHelp(false)` like picker; main `renderHelp()` keeps no results hints |

## Data Flow

    searchResultsMsg (multi) ─guard─→ resultsList.SetItems → mode=modeResults
                                            │
    KeyMsg ─modeResults→ updateResultsMode ─┤ Esc → modeNormal
                                            ├ Enter → queue.Add + modeNormal
                                            ├ a → openPlaylistPicker (pickerReturn=modeResults)
                                            ├ f → toggleFavorite (stay)
                                            └ up/down/j/k → resultsList.Update

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/ui/model.go` | Modify | Add `modeResults` const (end of mode iota); add `resultsList list.Model` field; build it in `New()` (delegate, `SetShowHelp(false)`, `SetShowStatusBar(false)`, `SetFilteringEnabled(false)`, Title "Resultados"); add `resultItem` adapter |
| `internal/ui/view.go` | Modify | Add `modeResults` branch at top of `View()`; remove `renderResults()` from main path + drop `JoinHorizontal`; render `renderQueue()` inline; delete `renderResults()` |
| `internal/ui/update.go` | Modify | Guard `searchResultsMsg` → populate `resultsList` + `modeResults`; route `modeResults` to new `updateResultsMode`; add `resultsList.SetSize` in `WindowSizeMsg`; ensure `urlResolvedMsg` stays normal (D1); `updatePickerMode` already restores `pickerReturn` (no change) |
| `internal/ui/keys.go` | None | Existing `Cancel`/`Enqueue`/`Up`/`Down`/`Favorite`/`AddToPlaylist` cover it |
| `internal/ui/update_test.go` | Modify | Keep `TestURLResolvedEnqueuesAndExposesResult` asserting `modeNormal` (D1); add modal tests |

## Interfaces / Contracts

```go
const ( /* … */ modeLyricsPicker; modeResults ) // appended to iota

type resultItem struct{ r search.Result; mark string } // mark = cacheMark prefix
func (i resultItem) Title() string       { return i.mark + i.r.Title }
func (i resultItem) Description() string { return i.r.Uploader }
func (i resultItem) FilterValue() string { return i.r.Title }
```

`searchResultsMsg` handler (guarded): build `[]list.Item` of `resultItem` (mark
via `m.cacheMark(r.ID)`), `m.resultsList.SetItems(items)`, `Select(0)`, and set
`modeResults` only from `modeNormal`/`modeSearch`. `updateResultsMode` mirrors
`updatePickerMode`: Esc → `modeNormal`; Enter → enqueue selected (reuse the
normal-mode enqueue/`started`/`loadTrackCmd` logic) + `modeNormal`; `a` →
`openPlaylistPicker(selected)` (sets `pickerReturn = modeResults`); `f` →
`toggleFavorite(selected)` and stay; default → delegate to `resultsList.Update`.
`WindowSizeMsg` adds `m.resultsList.SetSize(msg.Width, msg.Height-4)` beside the
picker. Modal footer renders a single help line shown only in `modeResults`
(e.g. "enter encolar · a +playlist · f favorito · ↑/↓ navegar · esc cerrar").

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | D1: URL resolve stays `modeNormal`, single result | Update existing `TestURLResolvedEnqueuesAndExposesResult` to assert `um.mode == modeNormal` |
| Unit | Multi-result opens modal | `searchResultsMsg{results: 2+}` from `modeNormal` → `modeResults`, items populated |
| Unit | Async guard | Same msg from `modeLibrary` → stays `modeLibrary`, no modal |
| Unit | Esc dismiss | `modeResults` + Esc → `modeNormal`, queue unchanged |
| Unit | Enter enqueue | `modeResults` + Enter → track in queue, `modeNormal` |
| Unit | `a` returns to modal | `a` → `modePicker` with `pickerReturn == modeResults`; pick → back to `modeResults`, no `refreshLibrary` |

Tests use `newTestModel` (`width=120,height=40`) so `resultsList` has non-zero
size. `D2` (entering search never reopens previous results) needs no new code —
`updateSearchMode` presents a fresh input, a non-empty submit runs a fresh
`doSearchCmd` that rebuilds `resultsList`, and an empty submit returns to
`modeNormal` without dispatching a search; assert no modal reopen on entering search.

## Migration / Rollout

No migration required. Single-PR, `internal/ui`-scoped; revert restores the
side-by-side panel.

## Open Questions

- [ ] Dismiss key: Esc is shared via `keys.Cancel`. Confirm Esc (no new binding)
      is the dismiss key for `modeResults` — recommended, matches pickers.
- [ ] `resultsList` delegate styling: reuse `list.NewDefaultDelegate()` (as
      `m.picker`) vs a custom delegate to surface the `⤓` cache mark colour.
      Recommend default delegate with mark inlined in `Title()`.
- [ ] List size: reuse `Height-4` (picker uses `-4`). Exploration noted "one more
      line"; recommend keeping `-4` for consistency unless the footer needs `-5`.

## Size Estimate vs Budget

~120-160 changed lines (model `resultItem`+field+`New()` ~25; view branch + strip
removal + `renderResults` deletion ~25; update guard + `updateResultsMode` +
`WindowSizeMsg` ~60; tests ~40). Comfortably under the 400-line review budget;
single PR.
