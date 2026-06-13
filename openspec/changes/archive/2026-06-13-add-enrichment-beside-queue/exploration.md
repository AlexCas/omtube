# Exploration: Add Enrichment Panels Beside Queue (Single Horizontal Row)

## Current State

### Main-view layout in `View()` (`internal/ui/view.go`)

The main TUI layout is built in `View()` using a `strings.Builder` (`b`). The relevant layout
writes are (line numbers reference the current file on branch `feat/enrichment-beside-queue`):

| Line | Code | Effect |
|------|------|--------|
| 52 | `b.WriteString(m.renderQueue())` | Writes the queue panel (Width 36) as its own row |
| 53 | `b.WriteString("\n")` | Newline → enrichment goes on the NEXT row |
| 57 | `b.WriteString(enrich)` (inside `if enrich := m.renderEnrichment(); enrich != ""`) | Writes lyrics+cover side-by-side but still in a SEPARATE row below the queue |
| 59 | `b.WriteString("\n")` | Trailing newline after enrichment row |

The result is a **two-row layout**:

```
Row 1: [ Queue panel (36 cols) ]
Row 2: [ Lyrics panel (50 cols) ][ Cover panel (28 cols) ]
```

### Panel dimensions

| Panel | Render function | Declared width (lipgloss) |
|-------|----------------|--------------------------|
| Queue | `renderQueue()` | `m.styles.panel.Width(36)` (line 144 / 166) |
| Lyrics | `renderLyricsPanel()` | `m.styles.panel.Width(50)` (line 361) |
| Cover | `renderArtworkPanel()` | `m.styles.panel.Width(28)` (line 407) |

Panels use `m.styles.panel` which applies a rounded border (`lipgloss.RoundedBorder()`, 1 cell
each side), so the rendered cell widths are approximately Width + 4 (2 border + 2 padding):

- Queue: ~40 rendered cols
- Lyrics: ~54 rendered cols
- Cover: ~32 rendered cols
- **Total when all three are present: ~126 rendered cols**

### `renderEnrichment()` (`view.go` lines 330–344)

```go
func (m Model) renderEnrichment() string {
    hasLyrics  := m.lyrics  != nil
    hasArtwork := m.artwork != nil
    if !hasLyrics && !hasArtwork {
        return ""
    }
    var panels []string
    if hasLyrics  { panels = append(panels, m.renderLyricsPanel()) }
    if hasArtwork { panels = append(panels, m.renderArtworkPanel()) }
    return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}
```

The function already does its own horizontal join and returns either a composed string or `""`.
The two enrichment services are independently optional: either, both, or neither may be enabled.

---

## Desired Outcome

A **single horizontal row**: `[ Queue | Lyrics | Cover ]` (or `[ Queue ]` when enrichment is off).

```
Row 1: [ Queue panel (36) ][ Lyrics panel (50) ][ Cover panel (28) ]
```

Rows below (`renderNowPlaying`, `renderHelp`, `renderVisualizer`) are unaffected.

---

## Affected Areas

- `internal/ui/view.go` — the only file that needs changes:
  - `View()`: replace the separate `b.WriteString(m.renderQueue())` + conditional
    `b.WriteString(enrich)` with a single composited horizontal row.
  - `renderEnrichment()`: optionally refactored to support the new composition, or kept as-is
    and called from `View()` differently (see Approaches below).
- `internal/ui/update_test.go` — review needed:
  - `TestRenderQueueWindowsLongQueue` (line 485): calls `m.renderQueue()` directly and asserts
    on its content (`Cola (100)`, `▼ 90 más`, newline count). This test does **NOT** assert on
    vertical positioning in `View()` — it is unaffected.
  - `TestToggleOffParity_NoEnrichmentPanels` (line 124): calls `m.View()` and checks that
    "Letra" and "Portada" are absent. The layout change does not alter when these strings appear
    — this test remains valid.
  - `TestLyricsPanel_*` and `TestArtworkPanel_*` (lines 148–195): call `renderLyricsPanel()` /
    `renderArtworkPanel()` directly. Unaffected.
  - No existing test asserts on vertical positioning or newline count of the full `View()` output,
    so the layout change does not break any test.

---

## Approaches

### Approach A — Compose in `View()` using `lipgloss.JoinHorizontal` directly

Modify `View()` to build the horizontal row inline:

```go
// In View():
enrichStr := m.renderEnrichment()
var mainRow string
if enrichStr != "" {
    mainRow = lipgloss.JoinHorizontal(lipgloss.Top, m.renderQueue(), enrichStr)
} else {
    mainRow = m.renderQueue()
}
b.WriteString(mainRow)
b.WriteString("\n")
```

- `renderEnrichment()` is unchanged — still returns `""` or `lipgloss.JoinHorizontal(lyrics, cover)`.
- The `if enrichStr != ""` guard avoids passing an empty string to `JoinHorizontal`, which could
  add trailing whitespace or misalign the queue panel.
- Pros: minimal diff (3–5 lines changed in `View()`), no change to `renderEnrichment()` contract,
  easy to understand.
- Cons: `renderQueue()` is called once inside `View()` rather than assigned to a variable first
  (minor style point — can assign).
- Effort: **Low**

### Approach B — Slice-based composition in `View()`

Collect all column panels into a slice and join once:

```go
// In View():
cols := []string{m.renderQueue()}
if enrich := m.renderEnrichment(); enrich != "" {
    cols = append(cols, enrich)
}
b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols...))
b.WriteString("\n")
```

- Equivalent to Approach A but uses a slice pattern that would scale if a third column were ever
  added.
- Pros: more extensible, slightly cleaner when more panels are added later.
- Cons: fractionally more code; `lipgloss.JoinHorizontal` with a single-element slice works
  correctly (returns the element as-is) but relying on that is implicit.
- Effort: **Low**

### Approach C — Refactor `renderEnrichment()` to accept the queue panel

Pass the queue panel into `renderEnrichment()` and have it always return the full row:

```go
func (m Model) renderMainRow() string {
    queue := m.renderQueue()
    hasLyrics  := m.lyrics  != nil
    hasArtwork := m.artwork != nil
    if !hasLyrics && !hasArtwork {
        return queue
    }
    var panels []string
    panels = append(panels, queue)
    if hasLyrics  { panels = append(panels, m.renderLyricsPanel()) }
    if hasArtwork { panels = append(panels, m.renderArtworkPanel()) }
    return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}
```

- Pros: all row-composition logic in one place, `View()` becomes cleaner.
- Cons: breaks the existing interface boundary (queue rendering entangled with enrichment
  logic); the current `renderEnrichment()` would become dead code or be removed, adding
  churn to an otherwise simple change. Tests call `renderLyricsPanel()` / `renderArtworkPanel()`
  directly, so those remain callable.
- Effort: **Low-Medium** (slightly more refactoring than A/B)

---

## Recommendation

**Approach A** is recommended. It is the smallest, most targeted change:
- 3–5 lines modified in `View()`.
- `renderEnrichment()` stays exactly as-is (contract unchanged, tests unchanged).
- The empty-enrichment guard (`if enrichStr != ""`) is explicit and easy to audit.
- Aligns perfectly with the user's decision: one horizontal row, no responsive fallback.

The conditional `if enrichStr != ""` before `JoinHorizontal` is the key correctness nuance:
`lipgloss.JoinHorizontal` with an empty string produces an empty (or whitespace-padded) cell
that would visually corrupt the queue panel alignment. The guard prevents this.

---

## Width Implications and Accepted Trade-off

With all three panels active:

| Panel | lipgloss Width | Estimated rendered cols (Width + 2 borders + 2 padding) |
|-------|----------------|----------------------------------------------------------|
| Queue | 36 | ~40 |
| Lyrics | 50 | ~54 |
| Cover | 28 | ~32 |
| **Total** | **114** | **~126** |

The `center()` function uses `lipgloss.PlaceHorizontal(m.width, lipgloss.Center, s)`. On terminals
narrower than ~126 columns, the row will overflow (horizontal truncation or wrapping depending on
the terminal emulator).

**This is a documented, accepted trade-off**: the user explicitly rejected responsive/width
fallback logic. The trade-off is noted here for the spec/design phase. No action is required.

---

## Risks

1. **Empty-enrichment spacing**: If `renderEnrichment()` returns `""` and it is naively passed
   to `JoinHorizontal`, lipgloss may render a phantom column with zero-width content but
   non-zero padding/border. The `if enrichStr != ""` guard in Approach A eliminates this.

2. **Vertical alignment mismatch**: When the queue panel has more rows than the enrichment panels
   (or vice versa), `lipgloss.JoinHorizontal(lipgloss.Top, ...)` aligns their top edges. The
   bottom of the shorter panel will be padded with empty space. This is correct and expected
   behavior — Lip Gloss handles this automatically. No action needed.

3. **Single-service enrichment**: If only lyrics is enabled (artwork nil) or only artwork
   (lyrics nil), `renderEnrichment()` still returns a non-empty string with just one panel.
   `JoinHorizontal(queue, onePanel)` works correctly in both cases.

4. **Horizontal overflow on narrow terminals**: Documented above. Accepted trade-off, not a
   blocker.

5. **No model/update changes**: The change is purely in the view layer. No state, no messages,
   no update logic touched.

---

## Open Questions for Human Gate

None blocking. The user has already made all key decisions:
- Layout: single horizontal row, always.
- No responsive fallback.
- Accepted width trade-off.

One informational question the orchestrator may surface:
- Should the `\n` separator between `renderQueue()` and the enrichment row (line 53 in `View()`)
  also be removed? Currently it creates a blank line between queue and enrichment — with the
  horizontal join this line goes away naturally (the join produces a single block). The answer
  is implicitly "yes" (it is replaced by the single-row join), but confirming alignment with
  the user's visual expectation during the proposal gate costs nothing.

---

## Ready for Proposal

**Yes.** The scope is clearly bounded, the change is low-effort, no model or update changes are
needed, and no existing tests will break. The proposal phase can immediately produce a concrete
diff plan.
