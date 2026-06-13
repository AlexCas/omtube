# Exploration: add-search-results-modal

## Current State

The main `View()` in `internal/ui/view.go` composes a single string from the
following sections, rendered top-to-bottom on every frame:

1. Title bar (`🎵 Omusic`)
2. Search input / status line
3. **Results + Queue panels side-by-side** (`lipgloss.JoinHorizontal` — the
   horizontal strip that is widest and contributes most to overflow)
4. Enrichment row: Lyrics panel (width 50) + Artwork panel (width 28), also
   side-by-side
5. Now-playing bar + progress
6. Help/instructions line
7. Animated bar visualizer

The results panel (`renderResults`, `view.go:117`) is 48 columns wide and
renders up to N results with a cursor. It is placed at `results[0]` of the
horizontal join; the queue panel is placed at `results[1]`, width 36. Combined,
the two-column row is ~90 columns wide, then the enrichment row (50+28 = ~84
columns + borders) stacks below it, making the total viewport height exceed
a standard 24-line terminal, especially when there are many results or lyric
lines.

**The results panel is always rendered**, even when there are no results and the
user is just listening to music. It shows "(vacío)" but still occupies space and
pushes subsequent panels down.

### Existing modal/takeover pattern

`view.go:20-22` shows the established pattern for full-screen takeover:

```go
if m.mode == modePicker || m.mode == modeLyricsPicker {
    return m.picker.View()
}
```

`modePicker` (add-to-playlist) and `modeLyricsPicker` (lyrics candidate
selection) both reuse a single `list.Model` (`m.picker`) and render it as the
sole content. When one of these modes is active, the entire viewport belongs to
the picker. `update.go:790-813` handles key routing for these modes.

There is no "floating overlay" compositing anywhere in the codebase — the
Lipgloss version in use (v1.1.0) does not provide a native overlay primitive.
All modal-like surfaces achieve isolation by returning an entirely different
string from `View()`.

## Affected Areas

- `internal/ui/model.go` — add a new `modeResults` constant; optionally add a
  dedicated `resultsList list.Model` (Approach A) or keep `m.results []search.Result`
  and the existing cursor (Approach B).
- `internal/ui/view.go` — remove `renderResults()` from the main `View()` path;
  add a takeover branch for `modeResults` (Approach A: `m.resultsList.View()`;
  Approach B: a thin manual renderer). Remove the `lipgloss.JoinHorizontal`
  of results+queue; keep queue inline.
- `internal/ui/update.go` — `searchResultsMsg` handler must transition to
  `modeResults` after populating results; add `updateResultsMode` to handle
  Esc (return to normal), Enter (enqueue + return), Up/Down (navigate),
  and `a`/`f` actions on the selected result.
- `internal/ui/keys.go` — no new bindings needed; existing `Cancel`, `Enqueue`,
  `Up`, `Down`, `Favorite`, `AddToPlaylist` already cover the required actions.
- `internal/ui/update_test.go` — existing `TestURLResolvedEnqueuesAndExposesResult`
  asserts on `um.mode`; will break if the mode is now `modeResults` after a URL
  resolve. Layout-sensitive tests (`TestRenderQueueWindowsLongQueue`) are
  unaffected because they test `renderQueue` in isolation. New tests needed for
  the results mode transitions.

## Approaches

### Approach A — Takeover modal backed by `bubbles/list`

Introduce a second `list.Model` (e.g. `m.resultsList`) alongside `m.picker`.
When search results arrive, populate `resultsList`, set `mode = modeResults`,
and in `View()` return `m.resultsList.View()` for this mode — exactly like
`modePicker`. Navigation, filtering, and keyboard handling are delegated to
the list component.

**Pros:**
- Mirrors the existing `modePicker` / `modeLyricsPicker` pattern exactly —
  minimal conceptual surface area for future contributors.
- Built-in filtering, scroll, and accessible keyboard nav from `bubbles/list`.
- `WindowSizeMsg` already calls `m.picker.SetSize(msg.Width, msg.Height-4)`;
  the same call covers `resultsList` with one more line.
- Consistent UX: all "choose one thing" surfaces look and behave the same.

**Cons:**
- Requires wrapping `search.Result` in a `list.Item` adapter (a small
  `resultItem` struct — identical effort to `playlistItem` / `candidateItem`
  already in `model.go:201-218`).
- The cache indicator (`⤓`) currently shown per row in `renderResults` must be
  surfaced via the `list.Item.Title()` or `Description()` methods rather than
  through `cacheMark()` directly; workable but slightly indirect.
- `bubbles/list` has its own key bindings that must not conflict with the
  existing `keyMap` (filter `/` key overlap potential — mitigated by disabling
  filtering on the list as done for `m.picker`: `picker.SetFilteringEnabled(false)`).

**Effort:** Low — ~60-80 lines changed/added, no new files.

---

### Approach B — Takeover modal backed by manual renderer

Introduce `modeResults` and render it via a thin custom function (similar to the
current `renderResults` but as a full-screen view). Keep `m.results []search.Result`
and `m.cursor int` as the state; add `updateResultsMode` in `update.go`.

**Pros:**
- Zero new struct types; re-uses existing `m.results` and cursor logic verbatim.
- The `cacheMark` helper integrates cleanly without adapters.
- Custom layout allows showing richer metadata (duration, uploader) without
  truncation constraints of `list.Item.Description()`.
- No risk of `bubbles/list` internal key routing interfering with the `keyMap`.

**Cons:**
- Duplicates scroll/windowing logic already implemented for the queue
  (`queueWindow`) — a new analogue for results must be written and tested.
- No built-in filtering: if filtering results in the modal is ever desired,
  it must be built from scratch.
- Slightly less consistent UX: the "pick from list" surfaces (picker, lyrics
  picker) use `bubbles/list` while results would use a hand-rolled renderer.

**Effort:** Low-Medium — ~80-100 lines changed/added.

---

### Approach C — Floating overlay (rejected)

Use Lipgloss 1.1.0 to draw a bordered box over the main view. **Not viable:**
Lipgloss 1.1.0 has no `Place`-over-string compositing for arbitrary overlays
without rewriting the rendering pipeline. The `lipgloss.Place` family positions
within a canvas, not on top of an existing string. The only way to fake a
floating overlay would be to measure and manually inject ANSI escape sequences —
fragile, untestable, and alien to the existing codebase style.

**Effort:** High; risk: High. Not recommended.

## Recommendation

**Approach A (takeover modal via `bubbles/list`)** is recommended.

It aligns perfectly with the existing `modePicker` + `modeLyricsPicker`
precedent, minimises the diff, and imports no new dependency (bubbles/list is
already in use). The cache indicator can be embedded in the `resultItem.Title()`
string. Filtering should be disabled on the results list to prevent the `/` key
from being swallowed inside the modal (same pattern as `m.picker`).

The main `View()` change collapses to: add one takeover branch for `modeResults`
at the top of `View()`, remove `results := m.renderResults()` and the
`JoinHorizontal`, and keep `renderQueue()` as an inline panel in the main view.

The `updateResultsMode` handler mirrors `updatePickerMode`:
- `Esc` → back to `modeNormal` (dismiss without action)
- `Enter` → enqueue selected + back to `modeNormal`
- `a` → open playlist picker for selected result (sets `pickerReturn = modeResults`)
- `f` → toggle favorite for selected result
- Up/Down/j/k → delegate to `m.resultsList.Update(msg)` for scroll

## Risks

1. **`pickerReturn` chaining**: `openPlaylistPicker` saves `m.pickerReturn = m.mode`
   so that `modePicker` knows where to return. With `modeResults` as the new
   caller, `updatePickerMode` must restore `modeResults` and NOT call
   `m.refreshLibrary()` (that branch is for `modeLibrary` callers only). This
   must be verified carefully to avoid the `a` key returning to the wrong mode.

2. **`searchResultsMsg` mode transition timing**: Currently the handler at
   `update.go:51-63` sets `m.results` and leaves the mode as `modeNormal`. The
   new code must also set `m.mode = modeResults`. If results arrive while the
   user has already navigated to a different mode (e.g., `modeLibrary` was open
   during an async search), the auto-transition would hijack the current view.
   A guard (`if m.mode == modeNormal || m.mode == modeSearch`) is needed.

3. **`urlResolvedMsg` mode**: `update.go:65-82` handles URL resolution; it
   populates `m.results` and keeps the mode normal. If the new convention is
   "always switch to `modeResults` when `m.results` is populated", this handler
   should also transition. Alternatively, URL resolution can skip the modal
   (results are a single auto-enqueued track). Decision needed.

4. **`WindowSizeMsg` coverage**: `m.resultsList.SetSize` must be added to the
   `tea.WindowSizeMsg` branch in `update.go:21-24`; otherwise the list renders
   at zero size until the window is resized.

5. **Empty-state on dismiss**: If the user dismisses the modal without picking,
   `m.results` still holds the last results. On the next `/` search, the main
   view (without the results panel) shows no leftover data — this is desirable
   and requires no extra cleanup. However, a status message should reflect that
   results are available but hidden (e.g. "5 results — press / to re-open").

6. **Test breakage**: `TestURLResolvedEnqueuesAndExposesResult` checks
   `um.mode` implicitly; `newTestModel` at `update_test.go:116` sets
   `m.width = 120, m.height = 40` which would be needed to avoid zero-size
   list rendering in tests. Existing tests for `renderResults` (none currently)
   are not an issue; but any new test for results mode must account for the
   `list.Model` needing non-zero size to render.

7. **Help line update**: The current `renderHelp()` string will need an entry
   for the results modal (e.g. "esc cerrar resultados" shown only in `modeResults`).
   This is cosmetic but affects the instructions line and the visualizer width
   calculation.

## Open Questions

1. **URL-resolved track modal**: Should a single URL-resolved track also open
   `modeResults`, or stay in `modeNormal` (since it is auto-enqueued, showing
   the modal for one item may feel redundant)?

2. **Re-open results**: Should the user be able to re-open the last results
   panel from `modeNormal` without re-running the search (e.g., pressing `/`
   with an empty input when `m.results` is non-empty)? Or is re-searching always
   required?

3. **`resultsList` vs shared `picker`**: Should the results takeover reuse
   `m.picker` (third reuse after playlist-picker and lyrics-picker) or have its
   own `list.Model`? Reusing `m.picker` avoids an extra field but couples three
   unrelated flows to the same component; a dedicated field is cleaner.

4. **Keyboard shortcut visibility**: The help line is already long. Should
   results-mode bindings be shown only when in `modeResults`, or should the
   main help line always advertise how to reach results?

## Ready for Proposal

Yes — the codebase has clear precedent for the takeover pattern, the scope is
well-bounded to `internal/ui` with no backend/playback changes, and the two
viable approaches are concrete enough for a proposal decision. Recommend the
orchestrator present Approach A (bubbles/list takeover) as the default with
the open questions above surfaced for human review before speccing.
