# Design: TUI Sidebar Redesign — Full-Height Sidebar + Main on Glass

## Technical Approach

Purely presentational change confined to `internal/ui/styles.go`, `internal/ui/view.go`,
`internal/ui/view_test.go`, and `internal/ui/testdata/*.golden`. `Model`, `Update`,
`messages`, `keys`, and services are untouched; no keybinding or behavior change. The
existing pure `layout` value (derived from `m.width`/`m.height` once per `View()`) gains
a top-level **sidebar / main** split (`sidebarW`/`mainW`/`sidebarH`/`mainH`) replacing the
queue+lyrics+artwork triad as the primary column division. The middle-section's
top-anchored band (`PlaceVertical(bodyH, Top, band)`, `view.go:527`) is replaced by
`JoinHorizontal(Top, sidebar, main)` where BOTH children are already `bodyH` tall — the
blank vertical band (rows 15–26 in `view_120x30.golden`, twelve blank rows) disappears by
construction. The 12/20 content caps are lifted so real content fills the body. Now-playing
is promoted to a bordered footer **card** (slice 2), which re-measures `chromeFixed`.
Delivered as 3 chained PRs, each < 400 changed lines (force-chained, 400-line budget).

The layout math derivations below are numbered `Dx` (design decisions) to match the prior
archived design's convention; the implementer builds to these numbers rather than
re-deriving them.

## Architecture Decisions

### Decision 1 (D1): Sidebar / main top-level split in `computeLayout`

The primary column split becomes **sidebar | main** instead of the old flat
queue | lyrics | artwork triad. The queue lives inside the sidebar; artwork+lyrics live
inside the main area. New `layout` fields:

```go
type layout struct {
    bp                      breakpoint
    sidebarW, mainW         int // OUTER widths of the two top-level columns (arg to Width())
    sidebarH, mainH         int // = bodyH; height each column box fills
    queueW                  int // inner queue content width (= sidebarW content, derived)
    lyricsW, artW           int // inner main content widths (artW 0 when hidden)
    progressW               int
    bodyH                   int
    maxQueueRows            int
    lyricWindow, plainLines int
    nowTitleTrunc           int
    libLineTrunc            int
    showArtwork             bool
    slimRail                bool // narrow (<90): sidebar collapsed to a rail
}
```

`usable = max(width-2, minUsable=40)` is unchanged. Both top-level columns are bordered
boxes, so each costs `panelBorder=2` columns outside its `Width()`. The split budget is
`split = usable - 2*panelBorder` (two outer borders), then:

- **Wide/medium (≥90):**
  - **D1a** `sidebarW = clamp(round(split * 0.30), sbMin=26, sbMax=40)`.
    30% lands the sidebar at ~34 cols on a 120-wide terminal, ~28 at 90 — enough for
    `Cola (N)` + a nav list + windowed rows without crowding the main area.
  - **D1b** `mainW = split - sidebarW`. Main takes the remainder (never exceeds `split`).
- **Narrow (<90) — slim rail (D1c):** `slimRail = true`;
  `sidebarW = clamp(round(split * 0.22), railMin=16, railMax=22)`;
  `mainW = split - sidebarW`. The rail holds a compact `Cola` heading + windowed queue
  (short/truncated titles); the main area keeps its maximum available width for lyrics.
  Artwork stays hidden (`showArtwork = bp != bpNarrow`, unchanged). No stacking, no full
  hide — matches the `Slim Rail at Narrow Width (@slice1)` requirement.

Boundary guarantee (D1d): `sidebarW + mainW + 2*panelBorder == usable` at every width, so
`sidebarW + mainW <= usable` holds at the retuned boundary widths (89/90, 119/120).

Inner content widths derive from the outer widths minus the box border+pad the renderer
adds (`- panelBorder`, i.e. `-2`, matching today's `renderQueueAt` etc.):

- **D1e** `queueW = sidebarW` (the queue panel IS the sidebar box; its heading/rows use
  `sidebarW-2` inner as today via `l.queueW-6` for the row prefix budget).
- **D1f** Within `mainW`, artwork and lyrics **stack** (artwork ABOVE lyrics — fixed
  decision), so they are NOT two side-by-side columns. Both use the full main inner width:
  `lyricsW = mainW`, and (when `showArtwork`) `artW = mainW`. The old three-way
  side-by-side clamp (`qMin/lMin/aMin/aMax`, folding remainder into lyrics) is **removed**
  — it no longer applies because artwork and lyrics share one column vertically. Keep only
  a lyrics floor: if `mainW-2 < lMin=28` at narrow, that is accepted (slim rail already
  reserves main's max width; `TestNoLineExceedsWidth` still guards overflow via truncation).

Rationale: a single sidebar/main split is the structural move that removes the blank band
(both children are `bodyH` tall) AND delivers the sidebar hierarchy. Percentage+clamp keeps
`sum ≤ usable` (no overflow) and non-collapse (mins). Stacking artwork above lyrics inside
one main column removes the third border column at medium/wide, giving lyrics full width.

### Decision 2 (D2): Heights flow into BOTH children; caps derive from body height

**D2a** `sidebarH = mainH = bodyH`. Each top-level column is rendered as a box exactly
`bodyH` rows tall (see D6 for the fill mechanism), so `JoinHorizontal(Top, sidebar, main)`
is exactly `bodyH` rows with no top-anchored remainder.

The old caps are **lifted** (this is the fix for the wasted band):

- **D2b `maxQueueRows`** — the clamp `clamp(bodyH-5, 3, 20)` loses its `20` ceiling.
  New: `maxQueueRows = clamp(sidebarH - queueChrome, 3, sidebarH)`, where
  `queueChrome = 5` = box border (2) + heading (1) + up to two `▲/▼ N más` markers (2),
  **plus the nav block in slice 2** (see D2e). In slice 1 (no nav yet) `queueChrome = 5`,
  so at 120×40 (`bodyH ≈ 26`, see arithmetic in D5) `maxQueueRows ≈ 21` vs the old 20 —
  the queue now grows with the terminal instead of stopping at 20.
- **D2c `lyricWindow`** — the clamp `clamp(bodyH-3, 3, 12)` loses its `12` ceiling.
  New: `lyricWindow = clamp(mainH - lyricChrome, 3, mainH)`, odd-normalized (keep the
  `if lyricWindow%2==0 { lyricWindow-- }` so the active line centers).
  `lyricChrome` = box border (2) + lyrics heading (1) + **artwork block height when
  `showArtwork`**. Artwork is rendered upstream at fixed 12 rows (+heading 1 = 13); so
  `lyricChrome = 3` when artwork hidden, `= 3 + 13 = 16` when artwork shown (D2f).
- **D2d `plainLines`** — same derivation as `lyricWindow` without odd-normalization:
  `plainLines = clamp(mainH - lyricChrome, 3, mainH)`.
- **D2e nav block (slice 2):** the sidebar nav header (Cola/Biblioteca/Favoritos/Historial)
  costs `navRows = 4` content rows + 1 accent-bar separator = 5 rows. When slice 2 lands,
  `queueChrome` becomes `5 + navRows(4) + sep(1) = 10`; re-derive `maxQueueRows =
  clamp(sidebarH - 10, 3, sidebarH)`. Slice 1 uses `queueChrome = 5` (no nav yet).
- **D2f artwork stack budget (slice 2):** when artwork stacks above lyrics, subtract its
  fixed 13 rows (12 art + 1 heading) from the lyrics window as in D2c. In slice 1 the main
  keeps the current lyrics|artwork side-by-side arrangement (intermediate) so
  `lyricChrome = 3`; slice 2 switches to the stacked layout and moves to `lyricChrome = 16`
  when `showArtwork`.

`minBody = 4` floor and the `clamp(..., 3, ...)` lower bounds are retained so nothing
collapses at 20 rows (the 60×20 mandatory-elements guarantee).

Rationale: deriving windows from `sidebarH`/`mainH` (which equal `bodyH`) makes content
consume the vertical space that `PlaceVertical(bodyH, Top, ...)` used to waste. Removing the
20/12 ceilings is the literal `Layout Resilience (@slice1 @slice2)` requirement ("former
fixed 12/20 caps MUST be raised or derived from `sidebarH`/`mainH`").

### Decision 3 (D3): `View` render pipeline — JoinHorizontal + footer card

**Slice 1 (`View` default branch), replacing lines 219–235:**

```
title (3 rows, bordered)                         -- unchanged
blank
now-playing bar (1 row)                          -- slice 1 keeps top bar (Option A intermediate)
blank
status / input (1 row)                           -- unchanged
blank
JoinHorizontal(Top, m.renderSidebar(l), m.renderMain(l))   -- bodyH rows, NO PlaceVertical
blank
help (helpRows)                                  -- unchanged
visualizer (1 row)                               -- unchanged
```

`renderMiddleSection`'s `return lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)`
(`view.go:527`) is deleted. `renderSidebar` and `renderMain` each return a `bodyH`-tall
box; the `JoinHorizontal` result is exactly `bodyH` rows. `renderMiddleSection`,
`renderEnrichment` are refactored into `renderSidebar`/`renderMain` (see D4).

**Slice 2 (footer card):** the top now-playing bar (rows "now-playing (1)" + its blank) is
removed from above the body; a bordered footer **card** is inserted between the body and the
help line:

```
title (3)
blank
status / input (1)
blank
JoinHorizontal(Top, sidebar, main)   -- bodyH rows
blank
footer card (4 rows: 2 border + 2 content)
blank
help (helpRows)
visualizer (1)
```

Card assembly (`renderNowPlayingCard(l)`): a `panel`-bordered box (rounded, accent border,
NO Background) of two content rows —
row 1: `state glyph (▶/⏸)  ·  truncated title`;
row 2: `progressBar  ·  pos/dur  ·  vol N`.
Positioned **below the body, above the visualizer** (`Footer Card (@slice2)` requirement).
The whole view block still passes through `m.center(...)` (horizontal only) unchanged.

### Decision 4 (D4): Renderers — `renderSidebar` / `renderMain`

**D4a `renderSidebar(l) string`** — builds a box `Width(l.sidebarW)` whose inner content is
forced to `sidebarH-2` rows (see D6). Content:
- Slice 1: `Cola (N)` heading + queue window (existing `renderQueueAt` body logic, moved
  in) + `▲/▼ N más` markers.
- Slice 2: an accent-bar nav header (Cola/Biblioteca/Favoritos/Historial, active accented
  via `navActive`, others `navItem`/muted) ABOVE the queue block, separated by a
  `sectionHeader` accent bar.
The historic no-arg `renderQueue()` wrapper (called by package tests) is retained and now
delegates to the queue-body portion.

**D4b `renderMain(l) string`** — builds a box `Width(l.mainW)` forced to `mainH-2` rows.
Content:
- Slice 1 (intermediate): the existing enrichment (lyrics [+ artwork side-by-side]) inside
  the main box.
- Slice 2: artwork STACKED above lyrics — `JoinVertical(Left, renderArtworkBlock(l),
  renderLyricsBlock(l))` — each with a `sectionHeader` accent-bar heading. Artwork block
  omitted when `!l.showArtwork`.
- Slice 3: when `m.mode == modeLibrary || modeCreatePlaylist`, the main box renders the
  library content (tabs, cursor list, library help) instead of artwork+lyrics (D8).

Historic wrappers `renderLyricsPanel()` / `renderArtworkPanel()` are retained (tests call
them); their `*At(l)` variants are re-scoped to `mainW` inner widths.

### Decision 5 (D5): `chromeFixed` re-measure (the arithmetic)

`chromeFixed` is the count of fixed (non-body, non-help) vertical rows in the default
`View`. It is re-measured, not guessed.

**Slice 1 (Option A intermediate — top bar retained):** the row structure is unchanged from
today, so `chromeFixed = 11` **stays**:
title(3) + blank(1) + nowplaying(1) + blank(1) + status(1) + blank(1) + blank-after-body(1)
+ help(measured separately) + viz(1) + trailing(1) = 11 fixed + helpRows.
`bodyH = max(height - (11 + helpRows(width)), minBody=4)`.

**Slice 2 (footer card):** the top now-playing bar (+its blank) is removed; the footer card
(4 rows) + its blank are added. Net delta:

    removed: nowplaying(1) + blank(1)        = -2
    added:   footer card(4) + blank(1)       = +5
    net                                       = +3

**D5a** `chromeFixed = 11 - 2 + 5 = 14` in slice 2.
`bodyH = max(height - (14 + helpRows(width)), minBody=4)`.

Worked values at 120 cols (helpRows(120) = 2, from the wrapped help in the current golden):
- 120×30 slice 1: `bodyH = 30 - (11+2) = 17`.
- 120×30 slice 2: `bodyH = 30 - (14+2) = 14`.
- 120×40 slice 1: `bodyH = 40 - (11+2) = 27`.
- 120×40 slice 2: `bodyH = 40 - (14+2) = 24`.
- 60×20 slice 2: helpRows(60) = 2 (help wraps at 60), `bodyH = 20 - (14+2) = 4 = minBody`.
  At the floor: sidebar box (2 border + heading + ≥1 queue row) and main box still render;
  the footer card (4 rows) + title + help + viz all remain present — 60×20 mandatory-element
  guarantee holds (see D9 clipping note).

### Decision 6 (D6): Making each column exactly `bodyH` tall

`JoinHorizontal(Top, a, b)` pads the SHORTER child with blank lines to the taller one's
height — it does NOT force either to `bodyH`. To guarantee both are `bodyH` and the join is
`bodyH` with no trailing whitespace band, each renderer wraps its content with
`lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, box)` on the already-bordered box, OR sets
`Height(bodyH)` on the box style (`panel.Width(w).Height(bodyH)`). **D6 choice:** use
`.Height(l.bodyH)` on the box style so the rounded border's bottom edge lands at row
`bodyH` (a visible full-height box, not whitespace padding). This is the concrete
mechanism behind "the sidebar border reaches the help line" from the exploration. Verified
against the `No blank vertical band at 120x40` scenario: with both boxes `Height(bodyH)`,
every body row belongs to a box, so no fully-blank row appears between body and help.

### Decision 7 (D7): Styles in `styles.go` (all translucent — foreground/border only)

New fields on the `styles` struct, all with NO `Background`:

```go
type styles struct {
    // ...existing: title, panel, heading, selected, current, dim, help, errorMsg, viz
    sidebar    lipgloss.Style // full-height box for the sidebar column (slice 1)
    card       lipgloss.Style // now-playing footer card box (slice 2)
    navActive  lipgloss.Style // active nav item (accent #e0aaff, bold) (slice 2)
    navItem    lipgloss.Style // inactive nav item (muted #a0a0a0) (slice 2)
    accentBar  lipgloss.Style // section-header accent bar/rule (accent #e0aaff) (slice 2)
}
```

- **D7a `sidebar` / `card`:** rounded border, `BorderForeground(#e0aaff)`, `Padding(0,1)`,
  NO `Background` — same shape as `panel` (the implementer may reuse `panel` for the
  sidebar box in slice 1 and add `card` in slice 2). `card` is the footer box.
- **D7b `navActive`:** `Foreground(#e0aaff).Bold(true)`; `navItem`: `Foreground(#a0a0a0)`.
  Active nav item accented, rest muted (`Active nav item is accented` scenario).
- **D7c `accentBar`:** `Foreground(#e0aaff)`; the `sectionHeader(label)` helper renders the
  heading text plus an accent-colored rule (e.g. `heading` + `\n` + `accentBar.Render("━"×n)`
  or a leading `▎`/`│` accent glyph). Foreground/border glyphs only, NO `Background`
  (`Section headers render with an accent bar` scenario).
- **D7d themed `bubbles/list` delegate for library-in-main (slice 3):** reuse the existing
  `caelestiaListDelegate()` (already foreground/border only) — but note library-in-main
  uses the cursor-list `renderLibList` path (`➤` prefix), NOT `bubbles/list`. No new opaque
  row bg is introduced. The `hasNoBackground` assert is extended to `sidebar`, `card`,
  `navActive`, `navItem`, `accentBar` and (unchanged) the delegate + list title.

`caelestiaListDelegate()` and `themedList()` are UNCHANGED — pickers/results stay
full-screen on `bubbles/list` (no `Update` coupling).

### Decision 8 (D8): Library into main (slice 3)

`View`'s `if m.mode == modeLibrary || modeCreatePlaylist { return m.renderLibrary(l) }`
(full-screen, `view.go:211-213`) is replaced by: build the same default block, but
`renderMain(l)` renders the **library content** (title/status handled by the outer chrome;
tabs `[Playlists] Favoritos Historial`, `renderLibList` cursor `➤`, library help) inside the
main box, while `renderSidebar(l)` persists with **"Biblioteca" accented** in the nav.
`renderLibrary`'s body logic is moved into a `renderLibraryInMain(l)` helper; the old
`renderLibrary` full-screen title/center chrome is dropped for these modes.
`modeResults`/`modePicker`/`modeLyricsPicker` branches (`view.go:193-205`) are UNCHANGED —
they stay full-screen (`Results modal and pickers stay full-screen` scenario). No `Update`
coupling. Delegate rows stay translucent.

## Data Flow

    m.width, m.height ──→ computeLayout() ──→ layout{sidebarW,mainW,sidebarH=mainH=bodyH,...}
                                                   │
                    ┌──────────────────────────────┴───────────────────────┐
                    ▼                                                        ▼
            renderSidebar(l)  [Height(bodyH)]                       renderMain(l)  [Height(bodyH)]
              nav header (s2) + queue window                          artwork block (s2, stacked)
                                                                      + lyrics window
                                                                      OR library content (s3)
                    └──────── JoinHorizontal(Top, sidebar, main) ───────────┘
                                          │  (exactly bodyH rows)
                     title ─ status ─ [body] ─ footerCard(s2) ─ help ─ viz ──→ center() ──→ View()

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/ui/view.go` | Modify | `layout` gains `sidebarW/mainW/sidebarH/mainH/slimRail`; `computeLayout` split (D1) + lifted caps (D2); `View` replaces `PlaceVertical` band with `JoinHorizontal(Top, sidebar, main)` (D3); new `renderSidebar`/`renderMain` (D4); footer card `renderNowPlayingCard` + re-measured `chromeFixed=14` (D5); `sectionHeader` helper (D7c); library-in-main (D8). |
| `internal/ui/styles.go` | Modify | Add `sidebar`, `card`, `navActive`, `navItem`, `accentBar` — all foreground/border, NO Background (D7). |
| `internal/ui/view_test.go` | Modify | Retune width/height asserts; extend `hasNoBackground`/palette asserts to new styles; add 120×40 case + no-blank-band + sidebar-persists asserts (see Testing Strategy). |
| `internal/ui/testdata/view_60x20.golden` `view_80x24.golden` `view_120x30.golden` | Modify | Regenerated per slice. |
| `internal/ui/testdata/view_120x40.golden` | Create | New tall fixture locking vertical fill (slice 1). |
| `internal/ui/testdata/view_library_120x30.golden` | Create | Library-in-main fixture (slice 3). |

## Interfaces / Contracts

New unexported symbols in `view.go` (no exported API, no `Model` field change):

```go
func (m Model) renderSidebar(l layout) string       // full-height sidebar box (slice 1)
func (m Model) renderMain(l layout) string           // full-height main box (slice 1)
func (m Model) renderNowPlayingCard(l layout) string // footer card (slice 2)
func (m Model) renderLibraryInMain(l layout) string  // library content in main (slice 3)
func sectionHeader(s styles, label string) string    // accent-bar heading (slice 2)
```

`renderMiddleSection`/`renderEnrichment` are removed or absorbed into
`renderSidebar`/`renderMain`. Historic no-arg wrappers `renderQueue`, `renderLyricsPanel`,
`renderArtworkPanel` (called by tests) are retained. `artworkWidth/artworkHeight` in
`update.go` stay 24×12 — NOT touched (outside the presentational surface).

## Slice Plan & Line Budget

| Slice | Scope | Files | Est. lines | Under 400? | Spec reqs |
|-------|-------|-------|-----------|-----------|-----------|
| **1 Structure** | `layout` new fields + `computeLayout` split (D1) + lifted 12/20 caps (D2, `queueChrome=5`, `lyricChrome=3`); replace `PlaceVertical` band with `JoinHorizontal(Top, sidebar, main)` both `Height(bodyH)` (D3/D6); `renderSidebar`/`renderMain` (D4, intermediate: keep lyrics+artwork side-by-side in main, now-playing stays top bar, `chromeFixed=11`); slim rail (D1c); retune `TestClassifyBoundaries`/`TestComputeLayoutWidths`/`TestComputeLayoutHeight`; regenerate 3 goldens + add 120×40. | `view.go`, `view_test.go`, 3 goldens mod + 1 new | ~250–340 | Yes | Sidebar+Main Layout; Layout Resilience; Slim Rail; Element Parity; Golden Determinism (120×40) — @slice1 |
| **2 Expressive styling** | `sectionHeader` accent bars (D7c); sidebar nav header active-accented (D7b, `queueChrome→10` D2e); artwork STACKED above lyrics in main (D2f/D4b, `lyricChrome→16`); now-playing → footer **card** (D3/D5a, `chromeFixed→14`); richer state colors; new styles (D7) all NO-Background; extend `hasNoBackground`+palette asserts. Regenerate 4 goldens. | `styles.go`, `view.go`, `view_test.go`, 4 goldens | ~270–380 | Yes (fallback below) | Now-Playing Footer Card; Accent-Bar Section Headers; Palette&Translucency; Element Parity — @slice2 |
| **3 Library in main** | `renderLibraryInMain` (D8): library content into main box, sidebar persists with "Biblioteca" accented; pickers/results stay full-screen; add library-in-main golden + sidebar-persists assert. Regenerate goldens. | `view.go`, `view_test.go`, +1 golden | ~160–250 | Yes | Library In Main; Element Parity; Golden Determinism — @slice3 |

**Re-slice fallback (forecast trigger):** if slice 2 forecasts > 400 changed lines at the
tasks phase, split into **2a** (accent-bar `sectionHeader` + sidebar nav header + palette
asserts; `queueChrome→10`; goldens) and **2b** (footer card + `chromeFixed→14` re-measure +
artwork-stacked `lyricChrome→16` + footer-clip asserts; goldens). 2a and 2b are each
independently shippable (2a leaves the top bar; 2b moves it to the card). No current
forecast crosses 400.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `classify` thresholds (unchanged 89/90, 119/120); `computeLayout` split: `sidebarW+mainW+2*panelBorder == usable`, mins hold, `slimRail` at <90, artwork hidden <90 | Retuned `TestClassifyBoundaries`, `TestComputeLayoutWidths` on 59/60/89/90/119/120 |
| Unit | Heights: `sidebarH==mainH==bodyH`; `maxQueueRows`/`lyricWindow`/`plainLines` grow with height (no 20/12 ceiling); odd `lyricWindow`; mins ≥3 | Retuned `TestComputeLayoutHeight` on 20/24/30/40; add 40-row grows-vs-30 assert |
| Golden | `View()` at 60×20, 80×24, 120×30, **120×40** | `compareGolden` + `UPDATE_GOLDEN=1 go test ./internal/ui`; review each `.got` |
| Invariant | No line exceeds width at 60×20 / 80×24 / 120×30 / **120×40** | Extend `TestNoLineExceedsWidth` size table with `{120,40}` |
| Invariant | **No blank vertical band** at 120×40 | New assert: render at 120×40, split body rows between title and help, assert no fully-blank row inside the `JoinHorizontal` region (every body row contains a box border/char) |
| Invariant | No `Background` on all new styles + delegate + list title | Extend `TestStylesNoBackground`/`hasNoBackground` to `sidebar`, `card`, `navActive`, `navItem`, `accentBar` (slice 2) |
| Invariant | Palette: accent `#e0aaff` on `navActive`/`accentBar`/`card` border; muted on `navItem` | Extend `TestCaelestiaAccentColors` (slice 2) |
| Invariant | Goldens differ: 80×24 ≠ 120×30 ≠ 120×40 pairwise | Extend `TestGoldensDiffer` names with `view_120x40.golden` |
| Parity | 60×20 mandatory elements present (title, now-playing/card, queue heading + ≥3 rows, help, viz); slim-rail artwork hidden + lyrics present | Extend `Test60x20NarrowNoArtwork` / add `TestFooterCardNoClip60x20` (slice 2) |
| Parity | Library-in-main: sidebar persists, "Biblioteca" accented, tabs + `➤` + help inside main; delegate/selection no-bg | New `TestLibraryInMainSidebarPersists` (slice 3) + keep `TestLibraryViewIsTranslucent` |
| Parity | `TestResultsModalGolden`, toggle-off parity tests unchanged | `go test ./internal/ui` |

Regenerate: `UPDATE_GOLDEN=1 go test ./internal/ui`. E2E: none (Go TUI — Playwright
disabled).

## Migration / Rollout

No migration. Each slice is an isolated PR touching only presentation files; reverting a
slice's commit leaves earlier slices valid (none touch `Model`/`Update`/services).
Chained-PR strategy `force-chained`, 400-line budget.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Golden churn masks a real regression | High | Regenerate per slice; lean on width/no-bg/palette/no-blank-band asserts, not only bytes; review each `.got` diff |
| `.Height(bodyH)` box + `JoinHorizontal` still leaves a whitespace row (D6) | Med | Assert no fully-blank body row at 120×40; if `.Height` under-fills, fall back to `PlaceVertical(bodyH, Top, box)` — both produce a `bodyH`-tall box |
| Footer-card `chromeFixed=14` clips a mandatory element at 20 rows (`bodyH=4` floor) | Med | Keep `minBody=4` + `clamp(...,3,...)` mins; `TestFooterCardNoClip60x20` asserts title, card, queue heading+≥3 rows, help, viz all present |
| Breakpoint edges (89/90, 119/120) break the split | Med | `sidebarW+mainW+2*panelBorder==usable` by construction (D1d); assert at each boundary in `TestComputeLayoutWidths` |
| Slim rail crowds lyrics at narrow | Med | Rail min 16 / max 22; main keeps remainder; assert no line exceeds width at 60/80 and lyrics present |
| Delegate/library-in-main re-adds opaque row bg | Med | Foreground/border only; library uses cursor-list `➤` path not `bubbles/list`; extend `hasNoBackground`; keep pickers on unchanged `caelestiaListDelegate()` |
| Slice 2 > 400 lines | Med | Pre-planned 2a/2b split; enforce at tasks |
| Artwork fixed 12 rows overflows `mainH` at 20 rows when stacked (D2f) | Med | When `mainH-2 < artHeight+lyricMin`, artwork block clips to available rows (heading + truncated art) before lyrics drop below 3; artwork already hidden <90 caps the case to medium/wide-tall |

## Edge-Case Arithmetic

- **Breakpoint 89/90:** at 89 `usable=87`, `split=83`, narrow slim rail
  `sidebarW=clamp(round(83*0.22),16,22)=18`, `mainW=65`. At 90 `usable=88`, `split=84`,
  wide/medium `sidebarW=clamp(round(84*0.30),26,40)=25→26` (clamped to `sbMin`),
  `mainW=58`, artwork returns (stacked). `sidebarW+mainW+4 == usable` at both.
- **Breakpoint 119/120:** at 119 `sidebarW=clamp(round(115*0.30),26,40)=35`, `mainW=80`.
  At 120 `usable=118`, `split=114`, `sidebarW=clamp(round(114*0.30),26,40)=34`, `mainW=80`.
  Both `+4 == usable`.
- **Footer card clip at 20 rows:** slice 2 `bodyH=max(20-(14+2),4)=4=minBody`. Body shows a
  4-row sidebar box (border 2 + heading 1 + 1 queue row via `maxQueueRows=clamp(4-10,3,4)=3`)
  and a 4-row main box; footer card (4) + title (3) + help (2) + viz (1) all render. No
  mandatory element dropped; over-long lines truncate (`TestNoLineExceedsWidth`).

## Open Questions

- [ ] None blocking. Exact split percentages (D1a 0.30, D1c 0.22) and rail min/max are
      tunable at apply time against the regenerated goldens; boundary widths (90, 120) and
      the `chromeFixed` arithmetic (11 → 14) are fixed by this design.
