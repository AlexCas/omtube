# Design: TUI Visual Redesign — Responsive Dashboard on Glass

## Technical Approach

Purely presentational change confined to `internal/ui/styles.go`, `internal/ui/view.go`,
`internal/ui/view_test.go`, and `internal/ui/testdata/*.golden`. `Model`, `Update`,
`messages`, `keys`, and services are untouched. A pure `layout` value is derived from
`m.width`/`m.height` once per `View()` and threaded into the render helpers, replacing
the hardcoded widths (36/50/28), progress bar (30), truncations (28/32/46/48/60),
`maxQueueRows` (10) and lyric windows (7/8). Delivered as 3 chained PRs, each < 400 lines.

## Architecture Decisions

### Decision 1: Breakpoint thresholds (columns)

The middle band today renders at ~118 cols: queue panel 36 + lyrics 50 + artwork 28,
plus 2 border/pad cols already inside each `Width(n)`. That is the source of the 80-col
overflow. Thresholds are set so the wide band only appears when all three panels fit.

**Choice**: `breakpoint(width int)` returns an enum:

| Breakpoint | Range (cols) | Columns shown | Width split (of usable = width − 2 outer margin) |
|------------|--------------|---------------|--------------------------------------------------|
| `bpNarrow` | `< 90`       | queue + lyrics (artwork HIDDEN) | queue ≈ 42%, lyrics ≈ 58% |
| `bpMedium` | `90–119`     | queue + lyrics + artwork | queue 34%, lyrics 40%, artwork 26% |
| `bpWide`   | `≥ 120`      | queue + lyrics + artwork | queue 30%, lyrics 44%, artwork 26% |

**Alternatives considered**: narrow `< 80` (rejected — at exactly 80 two full panels
+ borders still overflow; need headroom). Three columns in narrow (rejected — artwork
min legible width ~20 + 2 panels cannot coexist under 90).
**Rationale**: `< 90` guarantees two bordered panels fit at 80 (usable 78, split
33/45 incl. borders). Artwork returns at 90 where 3 minimal panels fit. `bpWide`
rebalances toward lyrics since artwork stays fixed-legible.

### Decision 2: No-`Background` assert mechanism

**Choice**: assert in Go via `lipgloss.Style.GetBackground()`. A helper
`hasNoBackground(s lipgloss.Style) bool` checks `s.GetBackground() == lipgloss.Color("")`
(the zero/no-color value). Applied to `styles.title` and `styles.panel` (Slice 1) and to
any themed `list` delegate styles (Slice 3).
**Alternatives considered**: SGR sequence probe on rendered output (grep for `48;` /
`\x1b[4…m`). Rejected — teatest/golden here render with ANSI stripped (plaintext), so
bg SGR is not observable in `View()` output; a probe would need a separate colored
render path and is brittle across lipgloss color profiles.
**Rationale**: `GetBackground()` inspects the style object directly, is profile-
independent, deterministic, and precisely encodes the requirement "style exposes no
Background".

### Decision 3: Fluid width formula

**Choice**: a `layout` struct computed once, with per-panel OUTER widths and derived
INNER widths (`inner = outer − 2` for border+pad). Truncations use inner widths.

```go
type layout struct {
    bp                       breakpoint
    queueW, lyricsW, artW    int // panel OUTER widths (0 = hidden)
    progressW                int // now-playing progress bar
    maxQueueRows             int
    lyricWindow, plainLines  int
    nowTitleTrunc            int
    libLineTrunc             int
    showArtwork              bool
}

func computeLayout(width, height int) layout
```

Widths: `usable = max(width-2, minUsable)`; each panel = `round(usable * pct)` clamped to
`[min, max]` (queue min 24 / lyrics min 28 / artwork fixed 24–28); remainder folded into
lyrics so the row fills `usable` without exceeding it. Inner truncations derive as
`panelW − 2`. Progress bar: `progressW = clamp(width − decorLen, 8, 40)` where `decorLen`
is the fixed now-playing chrome (state+title+times+vol ≈ 24 + title trunc). Visualizer
already sizes to `lipgloss.Width(help)` and stays as-is; help wraps/truncates to `width`.
**Alternatives considered**: fixed columns with a single narrow fallback (Option A floor).
Rejected — spec requires distinct medium/wide splits. **Rationale**: percentage + clamp
guarantees `sum ≤ usable` (no overflow) and non-collapse (mins), and makes 80 vs 120
differ.

### Decision 4: Vertical use (Slice 2)

**Choice**: chrome rows are measured, not guessed. `chrome = title(3) + gaps + nowPlaying(1)
+ status(1) + help(1) + visualizer(1)`. `bodyH = max(height − chrome, minBody)`. The middle
band is placed with `lipgloss.Place(width, bodyH, Center, Top, band)` so it occupies the
vertical slice instead of leaving dead rows; `PlaceVertical` centers the whole block when
`height` exceeds content. `maxQueueRows = clamp(bodyH − 2 (heading+borders), 3, 20)`;
`lyricWindow = clamp(bodyH − 2, 3, 12)` (odd-normalized so the active line centers);
`plainLines = lyricWindow`.
**Small heights (e.g. 20 rows)**: `minBody` floor (≈ 4 body rows) and the row mins keep
every MANDATORY element (title, now-playing, queue heading + ≥3 rows, help, visualizer)
visible; lyric/queue windows shrink first, artwork panel body shrinks to its heading +
clipped art before anything mandatory is dropped. Nothing mandatory is ever clipped.
**Alternatives considered**: keep `center()` (horizontal only). Rejected — Slice 2
requires height use. **Rationale**: `Place` is idiomatic lipgloss v1.1.0 and keeps the
block centered without touching content generation.

### Decision 5: Golden test strategy

**Choice**: extend `TestViewGolden` cases to `60×20`, `80×24`, `120×30`, regenerated with
`UPDATE_GOLDEN=1 go test ./internal/ui`. New assertions live in `view_test.go`:
- `TestNoLineExceedsWidth` (Slice 1): renders at 60/80/120, splits on `\n`, asserts
  `lipgloss.Width(line) <= width` for every line.
- `TestStylesNoBackground` (Slice 1): asserts `hasNoBackground(m.styles.title/panel)`.
- `TestGoldensDiffer` (Slice 1): asserts `view_80x24.golden != view_120x30.golden` bytes.
- `Test60x20NarrowNoArtwork` (Slice 2): asserts `!strings.Contains(out, "Portada")` and
  queue+lyrics present at 60×20.
- Slice 3: modal/picker no-`Background` assert on themed delegate styles.

80×24 and 120×30 WILL differ: at 80 → `bpNarrow` (2 cols, no artwork, queue≈32/lyrics≈44);
at 120 → `bpWide` (3 cols). Distinct column counts and widths guarantee byte-difference.
**Rationale**: byte goldens catch layout regressions; width/no-bg asserts catch the two
defects robustly regardless of golden churn (per proposal Risk table).

## Data Flow

    m.width, m.height ──→ computeLayout() ──→ layout{}
                                               │
        ┌──────────────────────────────────────┼───────────────┐
        ▼                     ▼                 ▼               ▼
    renderNowPlaying    renderQueue      renderLyricsPanel  renderArtworkPanel
    (progressW,          (queueW,         (lyricsW,          (artW; skipped if
     nowTitleTrunc)       maxQueueRows)    lyricWindow)       !showArtwork)
        └──────────── JoinHorizontal band ──→ Place(width,bodyH) ──→ View()

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/ui/styles.go` | Modify | Remove `Background(#1a1a2e)` at :21 (title) and :26 (panel); keep borders/accents. Slice 3: add themed delegate styles (foreground/border only). |
| `internal/ui/view.go` | Modify | Add `layout` struct + `breakpoint`/`computeLayout`; thread layout into `View`, `renderNowPlaying`, `renderQueue`, `renderLyricsPanel`, `renderArtworkPanel`, `renderMiddleSection`, `renderEnrichment`, `renderLibrary` (Slice 2 `Place`). |
| `internal/ui/view_test.go` | Modify | Add 60×20 case + 3 new asserts (Slice 1), narrow-artwork assert (Slice 2), delegate assert (Slice 3). |
| `internal/ui/testdata/view_60x20.golden` | Create | Narrow single-region fixture (Slice 2). |
| `internal/ui/testdata/view_80x24.golden` `view_120x30.golden` | Modify | Regenerated; now differ. |

## Interfaces / Contracts

New unexported symbols in `view.go` (no exported API change, no `Model` field change):

```go
type breakpoint int
const ( bpNarrow breakpoint = iota; bpMedium; bpWide )
func classify(width int) breakpoint
func computeLayout(width, height int) layout   // see Decision 3 struct
func hasNoBackground(s lipgloss.Style) bool     // test helper (view_test.go)
```

Render helpers gain a `l layout` parameter (e.g. `renderQueue(l layout) string`) instead
of reading package constants. `artworkWidth/artworkHeight` in `update.go` stay 24×12
(within `showArtwork`/`artW` budget) — not touched.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `classify` thresholds; `computeLayout` clamps sum ≤ usable, mins hold | Table test on boundary widths (59/60/89/90/119/120) |
| Golden | Rendered `View()` at 60×20, 80×24, 120×30 | `compareGolden` + `UPDATE_GOLDEN=1` |
| Invariant | No line exceeds width; no `Background`; 80≠120 goldens | New asserts in `view_test.go` |
| Parity | Existing tests (`TestToggleOffParity_*`) still pass unchanged | `go test ./internal/ui` |

E2E: none (TUI, no Playwright — `playwright_enabled: false`).

## Slice Plan & Line Budget

| Slice | Scope | Files | Est. lines | Under 400? |
|-------|-------|-------|-----------|-----------|
| 1 Base | Remove 2 bg fills; `layout`/`classify`/`computeLayout`; fluid widths+truncs from `m.width`; narrow breakpoint stops 80-col overflow (keep current vertical shape); width + no-bg + goldens-differ asserts; regenerate 80/120 goldens | `styles.go`, `view.go`, `view_test.go`, 2 goldens | ~180–240 | Yes |
| 2 Dashboard | Height use via `Place`/`PlaceVertical`; `maxQueueRows`/lyric windows from `bodyH`; medium/wide 3-col split; artwork hidden < 90; 60×20 golden + narrow-artwork assert | `view.go`, `view_test.go`, +1 golden, regen 2 goldens | ~200–300 | Yes |
| 3 Modals | Themed `bubbles/list` delegate (fg/border only) for `modeResults`/library/pickers; extend no-bg assert to delegate | `view.go`, `styles.go`, `view_test.go` | ~150–220 | Yes |

Re-slicing forecast: Slice 2 is the largest risk. If golden regeneration + the 3-column
`Place` math push it past 400, split into 2a (height/vertical fill only, keep 2-col) and
2b (medium/wide 3-col split + artwork breakpoint + 60×20 golden). No forecast currently
crosses the budget.

## Migration / Rollout

No migration required. Each slice is an isolated PR touching only presentation files;
reverting a slice's commit leaves earlier slices valid (none touch `Model`/`Update`/
services). Chained-PR strategy `ask-always` with 400-line budget.

## Open Questions

- [ ] None blocking. Exact percentages in Decision 1/3 are tunable at apply time against
      the regenerated goldens; boundary values (90, 120) are fixed by this design.
