# Tasks — TUI Sidebar Redesign: Chained Slices 1–3

**Change**: `tui-sidebar-redesign`
**Design ref**: `openspec/changes/tui-sidebar-redesign/design.md`
**Spec ref**: `openspec/changes/tui-sidebar-redesign/specs/caelestia-ui/spec.md` + `caelestia-ui.feature`
**Strategy**: force-chained, 400-line budget per slice.
**Playwright**: disabled (Go TUI; golden fixtures + asserts only).
**Scope**: `internal/ui/styles.go`, `internal/ui/view.go`, `internal/ui/view_test.go`,
  `internal/ui/testdata/*.golden`. No `Model`, `Update`, `messages`, `keys`, or services touched.

Tasks MUST be executed in numbered order within each slice.
Dependencies are noted inline. Each slice is independently shippable as a chained PR.

---

## Slice 1 — Structure

**Spec tags**: `@slice1`
**Estimated changed lines**: ~250–340 (code only; goldens are fixtures and do not count)
**Line budget check**: 340 < 400. If implementation exceeds 400, stop, report to orchestrator.
**Files**: `internal/ui/view.go`, `internal/ui/view_test.go`,
  modify `testdata/view_60x20.golden`, `testdata/view_80x24.golden`, `testdata/view_120x30.golden`,
  create `testdata/view_120x40.golden`.

### S1-T1 — Extend `layout` struct with sidebar/main fields (D1)

**File**: `internal/ui/view.go`
**Depends on**: nothing (first change of Slice 1)
**Spec mapping**: Sidebar + Main Layout (@slice1); Layout Resilience (@slice1); Slim Rail (@slice1)
**Design decisions**: D1, D2

- [x] S1-T1.1 — In `view.go`, add the following new fields to the `layout` struct
      (after the existing `bp` field, before `queueW`):
      ```go
      sidebarW, mainW   int  // outer widths for the two top-level columns (arg to Width())
      sidebarH, mainH   int  // = bodyH; height each column box fills
      slimRail          bool // true when bp == bpNarrow (sidebar collapses to a slim rail)
      ```
      Keep all existing fields (`bp`, `queueW`, `lyricsW`, `artW`, `progressW`, `bodyH`,
      `maxQueueRows`, `lyricWindow`, `plainLines`, `nowTitleTrunc`, `libLineTrunc`,
      `showArtwork`). The new fields are additive; no existing field is removed in this task.

- [x] S1-T1.2 — Verify `go build ./...` passes after the struct change.

Estimated lines changed: ~6 add.

---

### S1-T2 — Rewrite `computeLayout` to derive the sidebar/main split (D1, D2)

**File**: `internal/ui/view.go`
**Depends on**: S1-T1 (struct fields must exist)
**Spec mapping**: Sidebar + Main Layout (@slice1); Layout Resilience (@slice1); Slim Rail (@slice1)
**Design decisions**: D1a–D1f, D2a–D2d, D5 (chromeFixed stays 11 in slice 1)

- [x] S1-T2.1 — In `computeLayout`, REPLACE the existing three-way column budget block
      (the `if bp == bpNarrow { ... } else { ... }` block that sets `queueW`, `lyricsW`,
      `artW`, lines ~94–125 in the current file) with the new sidebar/main split:

      Wide/medium (bp != bpNarrow):
      ```
      split := usable - 2*panelBorder  // budget for two outer column boxes
      const sbMin, sbMax = 26, 40
      sidebarW = clamp(int(math.Round(float64(split)*0.30)), sbMin, sbMax)
      mainW = split - sidebarW
      slimRail = false
      ```
      Narrow (bp == bpNarrow) — slim rail (D1c):
      ```
      split := usable - 2*panelBorder
      const railMin, railMax = 16, 22
      sidebarW = clamp(int(math.Round(float64(split)*0.22)), railMin, railMax)
      mainW = split - sidebarW
      slimRail = true
      ```
      Boundary guarantee (D1d): `sidebarW + mainW + 2*panelBorder == usable` holds
      by construction because `mainW = split - sidebarW` and `split = usable - 2*panelBorder`.

- [x] S1-T2.2 — Derive inner content widths from the new split (D1e, D1f):
      ```
      queueW  = sidebarW   // sidebar box IS the queue panel (inner rows use sidebarW-2 via l.queueW-6)
      lyricsW = mainW      // main inner width; both artwork and lyrics share this column (stacked later)
      artW    = mainW      // same column; set to 0 when !showArtwork (narrow)
      ```
      In slice 1 (intermediate): keep `showArtwork = bp != bpNarrow` unchanged.
      When `bp == bpNarrow`, set `artW = 0`.

- [x] S1-T2.3 — Set the height fields from bodyH (D2a):
      ```
      sidebarH = bodyH
      mainH    = bodyH
      ```

- [x] S1-T2.4 — Lift the `maxQueueRows` ceiling (D2b). Replace:
      ```go
      maxQueueRows := clamp(bodyH-5, 3, 20)
      ```
      with:
      ```go
      const queueChrome = 5  // box border (2) + heading (1) + up to two ▲/▼ markers (2); slice 1 = no nav yet
      maxQueueRows := clamp(sidebarH-queueChrome, 3, sidebarH)
      ```
      This removes the `20` ceiling so the queue window grows with terminal height.

- [x] S1-T2.5 — Lift the `lyricWindow` ceiling (D2c). Replace:
      ```go
      lyricWindow := clamp(bodyH-3, 3, 12)
      ```
      with:
      ```go
      const lyricChrome = 3  // box border (2) + lyrics heading (1); slice 1 intermediate (no stacked artwork yet)
      lyricWindow := clamp(mainH-lyricChrome, 3, mainH)
      if lyricWindow%2 == 0 {
          lyricWindow--
      }
      ```

- [x] S1-T2.6 — Lift the `plainLines` ceiling (D2d). Replace:
      ```go
      plainLines: clamp(bodyH-3, 3, 12),
      ```
      with:
      ```go
      plainLines := clamp(mainH-lyricChrome, 3, mainH)
      ```

- [x] S1-T2.7 — Populate all new `layout` fields in the return literal:
      `sidebarW: sidebarW, mainW: mainW, sidebarH: sidebarH, mainH: mainH, slimRail: slimRail`.

- [x] S1-T2.8 — Keep `chromeFixed = 11` unchanged (slice 1 retains the top now-playing bar;
      `chromeFixed` is re-measured to 14 in slice 2 only). Do NOT change it here.

- [x] S1-T2.9 — Verify `go build ./...` passes.

Estimated lines changed: ~35–50 (replacing ~30 lines with ~35–50).

---

### S1-T3 — Add `renderSidebar` and `renderMain` renderers (D3, D4, D6)

**File**: `internal/ui/view.go`
**Depends on**: S1-T2 (layout fields must be populated)
**Spec mapping**: Sidebar + Main Layout (@slice1); Slim Rail (@slice1); Element Parity (@slice1)
**Design decisions**: D3, D4a, D4b (slice 1 intermediate), D6

- [x] S1-T3.1 — Add `func (m Model) renderSidebar(l layout) string`:
      - Build a bordered box `Width(l.sidebarW).Height(l.bodyH)` using the existing
        `panel` style (D7a note: the `sidebar` style is introduced in slice 2; slice 1
        reuses `panel` for the sidebar box).
      - Inner content: the `Cola (N)` heading + queue window (move the body logic from
        `renderQueueAt`) + `▲/▼ N más` markers. The historic `renderQueueAt` can be
        kept as a helper and called here.
      - The `.Height(l.bodyH)` on the box style is the D6 mechanism: the box bottom
        border lands at row `bodyH`, no trailing whitespace.
      - In slim-rail mode (`l.slimRail`): render a compact `Cola (N)` heading and the
        windowed queue with truncated titles (no change to content logic; the `queueW`
        is already narrow by D1c).

- [x] S1-T3.2 — Add `func (m Model) renderMain(l layout) string`:
      - Build a bordered box `Width(l.mainW).Height(l.bodyH)` using `panel` style.
      - Slice 1 intermediate: keep the existing lyrics + artwork side-by-side arrangement
        inside the main box (call `renderEnrichment(l)` for the inner content, scoped to
        `mainW` widths). Do NOT implement stacked artwork or footer card here — those are
        slice 2. Now-playing stays as the top bar.
      - `.Height(l.bodyH)` applied (D6).

- [x] S1-T3.3 — Retain all existing historic no-arg wrappers so package tests continue to
      compile and pass without changes:
      - `renderQueue()` — already delegates to `renderQueueAt`; keep as-is.
      - `renderLyricsPanel()` — keep delegating to `renderLyricsPanelAt`.
      - `renderArtworkPanel()` — keep delegating to `renderArtworkPanelAt`.

- [x] S1-T3.4 — Verify `go build ./...` passes.

Estimated lines changed: ~60–80 (two new renderer functions).

---

### S1-T4 — Replace `PlaceVertical` band with `JoinHorizontal` in `View` (D3, D6)

**File**: `internal/ui/view.go`
**Depends on**: S1-T3 (renderSidebar / renderMain must exist and compile)
**Spec mapping**: Sidebar + Main Layout (@slice1); No blank vertical band (@slice1)
**Design decisions**: D3, D6

- [x] S1-T4.1 — In `renderMiddleSection(l layout)`, DELETE the current body:
      ```go
      band := m.renderQueueAt(l)
      if enrich := m.renderEnrichment(l); enrich != "" {
          band = lipgloss.JoinHorizontal(lipgloss.Top, band, enrich)
      }
      return lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)
      ```
      REPLACE with:
      ```go
      sidebar := m.renderSidebar(l)
      main    := m.renderMain(l)
      return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
      ```
      Both `renderSidebar` and `renderMain` already return a `bodyH`-tall box (via
      `.Height(l.bodyH)` in S1-T3), so the join is exactly `bodyH` rows with no blank band.
      The `PlaceVertical` call at `view.go:527` is REMOVED.

- [x] S1-T4.2 — Add a guard in `renderMiddleSection` for the edge case where `l.mainW <= 0`
      (terminal so narrow that no main area fits): if `l.mainW <= 0`, return only
      `m.renderSidebar(l)` without joining. This prevents a zero-width `JoinHorizontal`
      panic at extreme sizes.

- [x] S1-T4.3 — Verify `go build ./...` passes.
- [x] S1-T4.4 — Verify `go test ./internal/ui -run TestNoLineExceedsWidth` still passes
      at 60, 80, 120 (pre-existing sizes) before adding the 120×40 case.

Estimated lines changed: ~10 (deletion + replacement).

---

### S1-T5 — Retune unit tests for the new `computeLayout` fields (D1, D2)

**File**: `internal/ui/view_test.go`
**Depends on**: S1-T4 (code must compile and behavior correct)
**Spec mapping**: Layout Resilience (@slice1); Slim Rail (@slice1); Golden Determinism (@slice1)
**Design decisions**: D1, D2

- [x] S1-T5.1 — In `TestClassifyBoundaries`: verify it still passes at 59/60/89/90/119/120
      (breakpoint thresholds are unchanged from the prior redesign). Run:
      `go test ./internal/ui -run TestClassifyBoundaries`. If it passes, no edit needed.

- [x] S1-T5.2 — Retune `TestComputeLayoutWidths` to assert the new split invariants.
      For each of the 6 boundary widths (59, 60, 89, 90, 119, 120) with height=24, assert:
      - `l.sidebarW + l.mainW + 2*panelBorder == usable` where `usable = max(width-2, 40)`
        (D1d boundary guarantee).
      - `l.sidebarW >= 16` (rail min) when `bp == bpNarrow`; `l.sidebarW >= 26` otherwise.
      - `l.mainW > 0` at all sizes.
      - `l.slimRail == (bp == bpNarrow)`.
      - Keep the existing `l.showArtwork == (bp != bpNarrow)` assert.
      - REMOVE the old `queueW + lyricsW + artW <= usable` three-way assert (no longer
        the primary split; the new invariant is `sidebarW + mainW + 2*panelBorder == usable`).

- [x] S1-T5.3 — Retune `TestComputeLayoutHeight` for the lifted caps.
      For heights [20, 24, 30, 40] with width=120, assert:
      - `l.sidebarH == l.bodyH` and `l.mainH == l.bodyH`.
      - `l.maxQueueRows >= 3` (floor holds).
      - `l.lyricWindow >= 3` and `l.lyricWindow%2 == 1` (odd-normalized).
      - At height=30: `l.maxQueueRows > 20` is now allowed (cap lifted). Assert
        `l.maxQueueRows > 12` (demonstrates the old 20-ceiling is gone at tall sizes).
      - At height=40: `l.maxQueueRows > l.lyricWindow` (queue can grow larger now).
      - REMOVE the old `< 20` ceiling asserts (the cap is gone in slice 1).
      - ADD: at height=40, `l.lyricWindow > 12` (demonstrates the old 12-ceiling is gone).

- [x] S1-T5.4 — Extend `TestNoLineExceedsWidth` to add the `{120, 40}` case:
      ```go
      {width: 120, height: 40},
      ```
      in the existing size table (alongside 60/80/120). Run after goldens regenerated (S1-T7).

- [x] S1-T5.5 — Add `TestNoBlankBodyBand(t *testing.T)`:
      Renders at 120×40, splits the output on `"\n"`, identifies the body region
      (rows between the status separator and the help line), and asserts that no fully-blank
      line appears inside the body region (every body row must contain at least one
      non-space character — a box border glyph or content).
      ```go
      func TestNoBlankBodyBand(t *testing.T) {
          m := newTestModel(t, Services{
              Lyrics:  fakeLyrics{},
              Artwork: fakeArtwork{art: "ART"},
          })
          m.width, m.height = 120, 40
          m.queue.Add(search.Result{ID: "a", Title: "Alpha Song", Uploader: "Alpha Artist"})
          m.curArtwork = "ASCII ART"
          out := m.View()
          lines := strings.Split(out, "\n")
          // Find the body region: skip chrome rows at top and bottom.
          // A safe heuristic: lines that are entirely whitespace within the middle
          // third of the output indicate the blank band. Assert none exist.
          bodyStart, bodyEnd := 7, len(lines)-5 // approximate chrome skip
          for i := bodyStart; i < bodyEnd && i < len(lines); i++ {
              if strings.TrimSpace(lines[i]) == "" {
                  t.Errorf("blank body row at line %d (0-indexed): body region has a blank band", i)
              }
          }
      }
      ```
      Note: the exact `bodyStart`/`bodyEnd` indices should be calibrated against the
      actual chrome structure during apply; the guard is that no fully-blank row appears
      in the body region.

- [x] S1-T5.6 — Add `TestGoldensDiffer` extension: after slice 1 regenerates `view_120x40.golden`,
      extend the test to assert `view_80x24`, `view_120x30`, and `view_120x40` are pairwise
      different (three `!bytes.Equal` asserts). If the function already exists, add the
      120×40 pair to it.

- [x] S1-T5.7 — Add the `{"120x40", 120, 40}` case to `TestViewGolden` in `view_test.go`
      so that `UPDATE_GOLDEN=1 go test ./internal/ui -run TestViewGolden` will create
      `testdata/view_120x40.golden`.

- [x] S1-T5.8 — Run all non-golden tests and confirm they pass before regenerating goldens:
      `go test ./internal/ui -run "TestClassifyBoundaries|TestComputeLayoutWidths|TestComputeLayoutHeight|TestNoBlankBodyBand|TestStylesNoBackground|TestCaelestiaAccentColors|TestDelegateNoBackground|TestLibraryViewIsTranslucent"`.

Estimated lines changed: ~90–110 (retunes + new asserts).

---

### S1-T6 — Regenerate goldens and add 120x40 fixture

**Files**: `internal/ui/testdata/view_60x20.golden`, `view_80x24.golden`, `view_120x30.golden`,
  `view_120x40.golden` (create)
**Depends on**: S1-T5 (all non-golden tests pass)
**Spec mapping**: Golden Determinism (@slice1); No blank vertical band (@slice1);
  Layout Resilience (@slice1)

- [x] S1-T6.1 — Run:
      `UPDATE_GOLDEN=1 go test ./internal/ui -run TestViewGolden`
      This regenerates the three existing goldens and creates `view_120x40.golden`.

- [x] S1-T6.2 — Inspect the diff of all four golden files deliberately:
      - `view_60x20.golden`: sidebar visible as slim rail; no artwork; lyrics present;
        no line exceeds 60 columns; no blank body rows.
      - `view_80x24.golden`: sidebar at narrow/medium width; no artwork; lyrics present.
      - `view_120x30.golden`: sidebar at wide width (~34 cols); main with artwork + lyrics;
        no blank body rows.
      - `view_120x40.golden` (new): confirms the body region is fully filled with sidebar
        and main boxes; NO fully-blank rows between title and help; sidebar border reaches
        help line.

- [x] S1-T6.3 — Run the full test suite (no UPDATE_GOLDEN):
      `go test ./internal/ui`
      Confirm all pass:
      - `TestViewGolden/60x20`, `/80x24`, `/120x30`, `/120x40` — PASS.
      - `TestNoBlankBodyBand` — PASS.
      - `TestNoLineExceedsWidth` (60/80/120/120×40) — PASS.
      - `TestGoldensDiffer` (all pairs) — PASS.
      - `TestClassifyBoundaries` — PASS.
      - `TestComputeLayoutWidths` — PASS.
      - `TestComputeLayoutHeight` — PASS.
      - `TestStylesNoBackground` — PASS (no regression).
      - `TestCaelestiaAccentColors` — PASS (no regression).
      - `TestDelegateNoBackground` — PASS (no regression).
      - `TestLibraryViewIsTranslucent` — PASS (no regression).
      - `TestResultsModalGolden` — PASS (no regression).
      - `Test60x20NarrowNoArtwork` — PASS (no regression).
      - `TestToggleOffParity_*` — PASS.

- [x] S1-T6.4 — Run `go test ./... && go vet ./...` — green.

- [x] S1-T6.5 — Confirm scope: only the following files were modified:
      `internal/ui/view.go`, `internal/ui/view_test.go`,
      `testdata/view_60x20.golden`, `testdata/view_80x24.golden`,
      `testdata/view_120x30.golden`, `testdata/view_120x40.golden`.
      No `model.go`, `update.go`, `messages.go`, `keys.go`, `styles.go`, or service file touched.
      No slice 2 or slice 3 code introduced.

- [x] S1-T6.6 — Count changed code lines (excluding goldens): confirm < 400.

Estimated: goldens are regenerated fixtures; not counted toward budget.

---

### Slice 1 — Summary table

| Task | Files | Est. lines | Design / Spec |
|------|-------|-----------|---------------|
| S1-T1 Extend layout struct | `view.go` | ~6 add | D1, D2 |
| S1-T2 Rewrite computeLayout | `view.go` | ~35–50 mod | D1a–f, D2a–d, D5 |
| S1-T3 renderSidebar / renderMain | `view.go` | ~60–80 add | D3, D4a/b, D6 |
| S1-T4 Replace PlaceVertical with JoinHorizontal | `view.go` | ~10 mod | D3, D6 |
| S1-T5 Retune tests + new asserts | `view_test.go` | ~90–110 add/mod | D1, D2, @slice1 |
| S1-T6 Regenerate goldens | `testdata/*.golden` | regen | Golden Determinism |

**Total code lines**: ~200–256 add/mod (< 400 budget).

### Slice 1 — Verification checklist (apply sign-off)

- [x] `go build ./...` — green
- [x] `go vet ./...` — no findings
- [x] `go test ./internal/ui` — all pass
- [x] `TestViewGolden/60x20`, `/80x24`, `/120x30`, `/120x40` — PASS
- [x] `TestNoBlankBodyBand` (120×40) — PASS
- [x] `TestNoLineExceedsWidth` (60/80/120/120×40) — PASS
- [x] `TestGoldensDiffer` (80×24 ≠ 120×30 ≠ 120×40 pairwise) — PASS
- [x] `TestClassifyBoundaries` — PASS
- [x] `TestComputeLayoutWidths` (sidebarW+mainW+4==usable at all 6 boundaries) — PASS
- [x] `TestComputeLayoutHeight` (caps lifted: >12 lyricWindow and >20 maxQueueRows at 40 rows) — PASS
- [x] `TestStylesNoBackground` — PASS (no regression)
- [x] `TestCaelestiaAccentColors` — PASS (no regression)
- [x] `test60x20NarrowNoArtwork` — PASS (no regression)
- [x] No `model.go`, `update.go`, `messages.go`, `keys.go` touched
- [x] No slice 2 or 3 code introduced
- [x] Changed lines < 400

---

## Slice 2 — Expressive Styling

**Spec tags**: `@slice2`
**Estimated changed lines**: ~270–380 (code only; goldens not counted)
**Line budget check**: 380 < 400. If implementation exceeds 400, apply the pre-planned
  2a/2b split (see section "Re-slice fallback 2a/2b" at the end of this slice).
**Files**: `internal/ui/styles.go`, `internal/ui/view.go`, `internal/ui/view_test.go`,
  modify all 4 goldens (`view_60x20.golden`, `view_80x24.golden`, `view_120x30.golden`,
  `view_120x40.golden`). Create `testdata/view_library_120x30.golden` is Slice 3 only.

**Ordering note**: S2-T1 (styles) must land before S2-T2 (view.go uses the new styles);
  S2-T3 and S2-T4 (footer card + chrome re-measure) must follow S2-T2 (nav header changes
  bodyH arithmetic); tests (S2-T5) must precede golden regeneration (S2-T6).

### S2-T1 — Add new styles to `styles.go` (D7)

**File**: `internal/ui/styles.go`
**Depends on**: nothing (first change of Slice 2)
**Spec mapping**: Caelestia Palette & Translucency (@slice2); Accent-Bar Section Headers (@slice2)
**Design decisions**: D7a, D7b, D7c

- [x] S2-T1.1 — Add the following new fields to the `styles` struct in `styles.go`:
      ```go
      sidebar   lipgloss.Style // full-height sidebar column box (slice 1 used panel; slice 2 gets own style)
      card      lipgloss.Style // now-playing footer card box
      navActive lipgloss.Style // active nav item (accent, bold)
      navItem   lipgloss.Style // inactive nav item (muted)
      accentBar lipgloss.Style // section-header accent bar/rule
      ```

- [x] S2-T1.2 — Populate the new styles in `defaultStyles()` (all NO `Background` — D7a–D7c):
      ```go
      sidebar: lipgloss.NewStyle().
          Border(lipgloss.RoundedBorder()).
          BorderForeground(lipgloss.Color("#e0aaff")).
          Padding(0, 1),
      card: lipgloss.NewStyle().
          Border(lipgloss.RoundedBorder()).
          BorderForeground(lipgloss.Color("#e0aaff")).
          Padding(0, 1),
      navActive: lipgloss.NewStyle().
          Foreground(lipgloss.Color("#e0aaff")).
          Bold(true),
      navItem: lipgloss.NewStyle().
          Foreground(lipgloss.Color("#a0a0a0")),
      accentBar: lipgloss.NewStyle().
          Foreground(lipgloss.Color("#e0aaff")),
      ```
      Verify: none of the five new styles calls `.Background(...)`.

- [x] S2-T1.3 — Verify `go build ./...` passes.

Estimated lines changed: ~25–30 add.

---

### S2-T2 — Add `sectionHeader` helper and sidebar nav header in `view.go` (D7c, D4a, D2e)

**File**: `internal/ui/view.go`
**Depends on**: S2-T1 (new style fields must exist)
**Spec mapping**: Accent-Bar Section Headers (@slice2); Active nav item (@slice2)
**Design decisions**: D7c, D4a (slice 2 nav header), D2e (queueChrome → 10)

- [x] S2-T2.1 — Add `func sectionHeader(s styles, label string) string` in `view.go`:
      Returns the label text rendered with `s.heading`, followed by a newline and an
      accent bar line rendered with `s.accentBar` (e.g. `s.accentBar.Render("━"×n)` or
      a leading accent glyph `▎`). Width `n` is passed as a parameter or derived from the
      label length. This is a pure formatting helper with no side effects.

- [x] S2-T2.2 — In `renderSidebar(l layout)`, ADD the nav header block ABOVE the queue:
      Nav items: `Cola`, `Biblioteca`, `Favoritos`, `Historial`.
      Active item (default: `Cola`) rendered with `m.styles.navActive`;
      inactive items with `m.styles.navItem`.
      Separate the nav block from the queue with a `sectionHeader` accent bar.
      The nav header occupies `navRows = 4` content rows + `sep = 1` accent-bar row = 5 rows total.
      Switch the sidebar box from `panel` style to `m.styles.sidebar`.

- [x] S2-T2.3 — Update `queueChrome` in `computeLayout` to account for the nav block (D2e).
      In `computeLayout`, change:
      ```go
      const queueChrome = 5
      maxQueueRows := clamp(sidebarH-queueChrome, 3, sidebarH)
      ```
      to:
      ```go
      const queueChrome = 10  // box border (2) + heading (1) + ▲/▼ (2) + nav rows (4) + sep (1)
      maxQueueRows := clamp(sidebarH-queueChrome, 3, sidebarH)
      ```

- [x] S2-T2.4 — Verify `go build ./...` passes.

Estimated lines changed: ~50–70 (helper ~15, nav header in renderSidebar ~25, queueChrome change ~5).

---

### S2-T3 — Stack artwork above lyrics in `renderMain`; update `lyricChrome` (D4b, D2f)

**File**: `internal/ui/view.go`
**Depends on**: S2-T2 (build green, sidebar nav complete)
**Spec mapping**: Sidebar + Main Layout — artwork stacked above lyrics (@slice2); Layout Resilience (@slice2)
**Design decisions**: D4b (slice 2), D2f

- [x] S2-T3.1 — In `renderMain(l layout)`, REPLACE the intermediate slice 1 content
      (which called `renderEnrichment` with side-by-side arrangement) with the stacked layout:
      ```go
      var parts []string
      if l.showArtwork {
          parts = append(parts, sectionHeader(m.styles, "Portada"))
          parts = append(parts, m.renderArtworkPanelAt(l))
      }
      parts = append(parts, sectionHeader(m.styles, "Letra"))
      parts = append(parts, m.renderLyricsPanelAt(l))
      inner := lipgloss.JoinVertical(lipgloss.Left, parts...)
      return m.styles.sidebar.Width(l.mainW).Height(l.bodyH).Render(inner)
      ```
      Note: use `m.styles.card` for the main box or `panel` — whichever the implementer
      decides at apply time; the key requirement is `Width(l.mainW).Height(l.bodyH)` and
      NO `Background`.

- [x] S2-T3.2 — Update `lyricChrome` in `computeLayout` for the stacked layout (D2f):
      In `computeLayout`, change `lyricChrome`:
      ```go
      // Slice 2: artwork stacks above lyrics. When showArtwork, subtract artwork block
      // (12 art rows + 1 heading = 13) from the lyrics window.
      lyricChrome := 3 // base: box border (2) + lyrics heading (1)
      if showArtwork {
          lyricChrome += 13 // artwork block: 12 rows of art + 1 heading
      }
      lyricWindow := clamp(mainH-lyricChrome, 3, mainH)
      if lyricWindow%2 == 0 {
          lyricWindow--
      }
      plainLines := clamp(mainH-lyricChrome, 3, mainH)
      ```

- [x] S2-T3.3 — Verify `go build ./...` passes.

Estimated lines changed: ~30–40 (renderMain rewrite ~20, lyricChrome update ~15).

---

### S2-T4 — Now-playing footer card and `chromeFixed` re-measure (D3, D5a)

**File**: `internal/ui/view.go`
**Depends on**: S2-T3 (artwork stack in place, build green)
**Spec mapping**: Now-Playing Footer Card (@slice2); Element Parity (@slice2)
**Design decisions**: D3 (slice 2 layout), D5a

- [x] S2-T4.1 — Add `func (m Model) renderNowPlayingCard(l layout) string`:
      Returns a 4-row bordered box (`m.styles.card.Width(l.progressW + l.nowTitleTrunc + 10)`
      or full width — calibrate at apply time). Content:
      - Row 1: state glyph (`▶` or `⏸`) + `·` + `truncate(m.curTrackTitle, l.nowTitleTrunc)`.
      - Row 2: `progressBar(m.pos, m.dur, l.progressW)` + `·` + `pos/dur time` + `·` + `vol N`.
      NO `Background`. Border uses `m.styles.card`.

- [x] S2-T4.2 — In `View()` (the default branch), remove the top now-playing bar and its
      blank separator (the `m.renderNowPlaying(l)` call and the preceding/following blank
      lines). Insert the footer card and its blank AFTER the body block and BEFORE the help
      line:
      ```go
      // REMOVE from top area:
      //   b.WriteString(m.renderNowPlaying(l))
      //   b.WriteString("\n")

      // KEEP body:
      b.WriteString(m.renderMiddleSection(l))
      b.WriteString("\n")

      // ADD footer card:
      b.WriteString(m.renderNowPlayingCard(l))
      b.WriteString("\n")

      // KEEP help:
      b.WriteString(m.renderHelp())
      ```

- [x] S2-T4.3 — In `computeLayout`, update `chromeFixed` to 14 (D5a):
      ```go
      // Slice 2: top now-playing bar removed (-2 rows), footer card added (+5 rows). Net +3.
      // chromeFixed = 11 - 2 + 5 = 14.
      const chromeFixed = 14
      ```
      The `bodyH` derivation (`max(height-(chromeFixed+helpRows(width)), minBody)`) then
      automatically yields the correct values:
      - 120×30 slice 2: `bodyH = 30 - (14+2) = 14`.
      - 120×40 slice 2: `bodyH = 40 - (14+2) = 24`.
      - 60×20 slice 2: `bodyH = 20 - (14+2) = 4 = minBody`. Floor holds.

- [x] S2-T4.4 — Verify `go build ./...` passes.

Estimated lines changed: ~50–70 (new renderNowPlayingCard ~30, View edits ~15, chromeFixed ~5).

---

### S2-T5 — Extend tests for slice 2 (no-bg asserts, palette, footer card)

**File**: `internal/ui/view_test.go`
**Depends on**: S2-T4 (all slice 2 code in place, build green)
**Spec mapping**: Caelestia Palette & Translucency (@slice2); Now-Playing Footer Card (@slice2);
  Accent-Bar Section Headers (@slice2); Element Parity (@slice2)
**Design decisions**: D7

- [x] S2-T5.1 — Extend `TestStylesNoBackground` to assert no `Background` on all 5 new styles:
      ```go
      if !hasNoBackground(s.sidebar)   { t.Error("sidebar must have no Background") }
      if !hasNoBackground(s.card)      { t.Error("card must have no Background") }
      if !hasNoBackground(s.navActive) { t.Error("navActive must have no Background") }
      if !hasNoBackground(s.navItem)   { t.Error("navItem must have no Background") }
      if !hasNoBackground(s.accentBar) { t.Error("accentBar must have no Background") }
      ```

- [x] S2-T5.2 — Extend `TestCaelestiaAccentColors` to assert the 5 new styles
      carry the correct palette colors:
      - `s.sidebar.GetBorderTopForeground() == lipgloss.Color("#e0aaff")`.
      - `s.card.GetBorderTopForeground() == lipgloss.Color("#e0aaff")`.
      - `s.navActive.GetForeground() == lipgloss.Color("#e0aaff")`.
      - `s.navItem.GetForeground() == lipgloss.Color("#a0a0a0")`.
      - `s.accentBar.GetForeground() == lipgloss.Color("#e0aaff")`.

- [x] S2-T5.3 — Add `TestFooterCardNoClip60x20(t *testing.T)`:
      Renders at 60×20 with footer card present (slice 2 `chromeFixed=14`, `bodyH=4=minBody`).
      Asserts all mandatory elements are present in the output:
      - Title line contains `Omusic` (or the 🎵 emoji).
      - Footer card content: state glyph `▶` or `⏸`, and `vol` keyword.
      - Queue heading: `Cola`.
      - Help text present (any help keyword, e.g. `buscar`).
      - Visualizer line present (at least one `▁▂▃▄` character or the flat char).
      - No line exceeds 60 columns (`lipgloss.Width(line) <= 60` for all lines).

- [x] S2-T5.4 — Run all non-golden tests:
      `go test ./internal/ui -run "TestStylesNoBackground|TestCaelestiaAccentColors|TestFooterCardNoClip60x20|TestNoLineExceedsWidth|TestNoBlankBodyBand|TestComputeLayoutHeight|TestComputeLayoutWidths"`.
      All must pass before regenerating goldens.

Estimated lines changed: ~60–80 add.

---

### S2-T6 — Regenerate all 4 goldens

**Files**: `testdata/view_60x20.golden`, `view_80x24.golden`, `view_120x30.golden`,
  `view_120x40.golden`
**Depends on**: S2-T5 (all non-golden tests pass)
**Spec mapping**: Golden Determinism (@slice1 @slice2)

- [x] S2-T6.1 — Run:
      `UPDATE_GOLDEN=1 go test ./internal/ui -run TestViewGolden`

- [x] S2-T6.2 — Inspect the diff of all four golden files deliberately:
      - Footer card visible below the body and above the visualizer in all sizes.
      - Top now-playing bar NO LONGER present.
      - Nav header (Cola / Biblioteca / Favoritos / Historial) visible in sidebar.
      - Artwork stacked above lyrics in the main area at 80×24 and 120×30/40 (wide).
      - No line exceeds the target width.
      - `view_60x20.golden`: all mandatory elements fit at 4-row body.

- [x] S2-T6.3 — Run the full test suite:
      `go test ./internal/ui`
      All tests pass, including:
      - `TestViewGolden` (all 4 sizes) — PASS.
      - `TestFooterCardNoClip60x20` — PASS.
      - `TestNoBlankBodyBand` — PASS.
      - `TestGoldensDiffer` (pairwise) — PASS.
      - `TestStylesNoBackground` (including new 5 styles) — PASS.
      - `TestCaelestiaAccentColors` (including new 5 styles) — PASS.

- [x] S2-T6.4 — Run `go test ./... && go vet ./...` — green.

- [x] S2-T6.5 — Confirm scope: only `styles.go`, `view.go`, `view_test.go`, and the 4 goldens
      were modified. No `model.go`, `update.go`, `messages.go`, `keys.go`, or service file touched.
      No slice 3 code introduced.

- [x] S2-T6.6 — Count changed code lines (excluding goldens): confirm < 400.

---

### Re-slice fallback 2a/2b (trigger if forecast > 400)

If the implementer forecasts that the total code lines in S2-T1 through S2-T5 will exceed
400 changed lines, apply the following pre-planned split before starting apply:

**Slice 2a** (accent headers + nav; stops before footer card):
- S2-T1 (new styles in styles.go)
- S2-T2 (sectionHeader helper + sidebar nav header; queueChrome → 10)
- S2-T5.1 + S2-T5.2 (hasNoBackground + palette asserts for new styles only)
- S2-T6 scoped to 4 goldens reflecting the nav header (no footer card yet; top bar still present)

**Slice 2b** (footer card + chrome re-measure + stacked artwork):
- S2-T3 (stacked artwork in renderMain; lyricChrome → 16 when showArtwork)
- S2-T4 (renderNowPlayingCard; chromeFixed → 14; remove top bar)
- S2-T5.3 + S2-T5.4 (TestFooterCardNoClip60x20)
- S2-T6 (regenerate all 4 goldens with footer card)

Both 2a and 2b are independently shippable (2a leaves the top bar; 2b moves it to the card).
No current forecast crosses 400 — this split is a contingency only.

---

### Slice 2 — Summary table

| Task | Files | Est. lines | Design / Spec |
|------|-------|-----------|---------------|
| S2-T1 New styles (sidebar/card/navActive/navItem/accentBar) | `styles.go` | ~25–30 add | D7a–c, @slice2 |
| S2-T2 sectionHeader + nav header; queueChrome→10 | `view.go` | ~50–70 add/mod | D7c, D4a, D2e |
| S2-T3 Stacked artwork in renderMain; lyricChrome→16 | `view.go` | ~30–40 mod | D4b, D2f |
| S2-T4 renderNowPlayingCard; chromeFixed→14; remove top bar | `view.go` | ~50–70 add/mod | D3, D5a |
| S2-T5 Extend tests (no-bg, palette, footer clip) | `view_test.go` | ~60–80 add | D7, @slice2 |
| S2-T6 Regenerate 4 goldens | `testdata/*.golden` | regen | Golden Determinism |

**Total code lines**: ~215–290 add/mod (< 400 budget).

### Slice 2 — Verification checklist (apply sign-off)

- [x] `go build ./...` — green
- [x] `go vet ./...` — no findings
- [x] `go test ./internal/ui` — all pass
- [x] `TestViewGolden` (60×20, 80×24, 120×30, 120×40) — PASS
- [x] `TestFooterCardNoClip60x20` — PASS (all mandatory elements at 4-row body)
- [x] `TestNoBlankBodyBand` — PASS (no regression)
- [x] `TestNoLineExceedsWidth` (all 4 sizes) — PASS
- [x] `TestGoldensDiffer` (pairwise) — PASS
- [x] `TestStylesNoBackground` (5 new styles included) — PASS
- [x] `TestCaelestiaAccentColors` (5 new styles included) — PASS
- [x] `TestDelegateNoBackground` — PASS (no regression)
- [x] `TestLibraryViewIsTranslucent` — PASS (no regression)
- [x] `TestResultsModalGolden` — PASS (no regression)
- [x] `Test60x20NarrowNoArtwork` — PASS (no regression)
- [x] `TestComputeLayoutHeight` (updated for chromeFixed=14) — PASS
- [x] `TestComputeLayoutWidths` — PASS (no regression)
- [x] Footer card visible in all 4 goldens; top now-playing bar absent
- [x] Nav header (Cola / Biblioteca / Favoritos / Historial) visible in sidebar goldens
- [x] No `model.go`, `update.go`, `messages.go`, `keys.go` touched
- [x] No slice 3 code introduced
- [x] Changed code lines < 400

---

## Slice 3 — Library in Main

**Spec tags**: `@slice3`
**Estimated changed lines**: ~160–250 (code only; goldens not counted)
**Line budget check**: 250 < 400.
**Files**: `internal/ui/view.go`, `internal/ui/view_test.go`,
  create `testdata/view_library_120x30.golden`.
  No changes to `styles.go` (all required styles already exist from slices 1–2).

**Ordering note**: S3-T1 (renderLibraryInMain) must precede S3-T2 (View routing); S3-T3 (tests)
  must precede S3-T4 (golden creation).

### S3-T1 — Add `renderLibraryInMain` helper (D8)

**File**: `internal/ui/view.go`
**Depends on**: nothing (first change of Slice 3; slices 1 and 2 must be merged)
**Spec mapping**: Library In Main (@slice3); Element Parity (@slice3)
**Design decisions**: D8

- [x] S3-T1.1 — Add `func (m Model) renderLibraryInMain(l layout) string`:
      Moves the body content from the existing `renderLibrary(l layout)` into the main box:
      - Renders the library tabs (`[Playlists]`, `Favoritos`, `Historial`) with active
        bracketed, using the `renderLibList` path (cursor `➤`).
      - Renders the create-playlist prompt when in `modeCreatePlaylist`.
      - Renders the library help line.
      - Returns the content string (NOT yet wrapped in a box — the caller wraps it in
        `renderMain`'s box).
      - Delegate rows use the existing `m.styles.selected` / `m.styles.dim` path — no
        new `Background` introduced.

- [x] S3-T1.2 — In `renderSidebar(l layout)`, when `m.mode == modeLibrary ||
      m.mode == modeCreatePlaylist`, render "Biblioteca" in `m.styles.navActive` and
      the other nav items in `m.styles.navItem` (instead of Cola accented).
      This is the "Biblioteca accented in nav" behavior required by the spec.

- [x] S3-T1.3 — Verify `go build ./...` passes.

Estimated lines changed: ~60–80 add.

---

### S3-T2 — Route library modes into `renderMain` in `View` (D8)

**File**: `internal/ui/view.go`
**Depends on**: S3-T1 (renderLibraryInMain must exist and compile)
**Spec mapping**: Library In Main (@slice3); Results modal full-screen (@slice3)
**Design decisions**: D8

- [x] S3-T2.1 — In `renderMain(l layout)`, ADD a branch at the top:
      ```go
      if m.mode == modeLibrary || m.mode == modeCreatePlaylist {
          inner := m.renderLibraryInMain(l)
          return m.styles.sidebar.Width(l.mainW).Height(l.bodyH).Render(inner)
      }
      // ... existing artwork+lyrics stacked content
      ```
      This replaces the old full-screen `renderLibrary` return for these two modes.

- [x] S3-T2.2 — In `View()`, REMOVE (or comment with a note) the existing early-return
      branch that short-circuits to `renderLibrary` for `modeLibrary`/`modeCreatePlaylist`
      (currently at `view.go:211-213` approximately):
      ```go
      // REMOVE: if m.mode == modeLibrary || m.mode == modeCreatePlaylist { return m.renderLibrary(l) }
      ```
      The library content now flows through the same default `View` block, with the sidebar
      always rendered and `renderMain` routing the content.

- [x] S3-T2.3 — Confirm the `modeResults`, `modePicker`, and `modeLyricsPicker` branches in
      `View()` (around `view.go:193-205`) are UNCHANGED — they remain full-screen returns.
      Do NOT move them into the sidebar+main body. Add a brief comment if needed to signal
      these stay full-screen intentionally.

- [x] S3-T2.4 — Verify `go build ./...` passes.

Estimated lines changed: ~25–35 mod.

---

### S3-T3 — Add `TestLibraryInMainSidebarPersists` and extend no-bg asserts

**File**: `internal/ui/view_test.go`
**Depends on**: S3-T2 (library routing complete, build green)
**Spec mapping**: Library In Main (@slice3); Element Parity (@slice3)
**Design decisions**: D8, D7d

- [x] S3-T3.1 — Add `TestLibraryInMainSidebarPersists(t *testing.T)`:
      Constructs a model with `m.mode = modeLibrary`, `m.width = 120`, `m.height = 30`,
      at least 2 entries in `m.libFavorites`. Calls `m.View()`. Asserts:
      - The output contains `Cola` (sidebar persists — nav item visible).
      - The output contains `Biblioteca` (the active nav item is accented in the sidebar).
      - The output contains `➤` (library cursor inside main area).
      - The output contains at least one tab name (`Playlists`, `Favoritos`, or `Historial`).
      - No line exceeds 120 columns.
      - `hasNoBackground(m.styles.selected)` — delegate rows stay translucent.

- [x] S3-T3.2 — Confirm `TestLibraryViewIsTranslucent` (from prior slice 3 of tui-visual-redesign)
      still passes with the new routing. If the test was checking the OLD full-screen
      `renderLibrary`, update it to match the new in-main rendering path:
      - If it uses `m.View()` and checks for content + cursor + no-bg on styles, it
        should continue to pass without change. Verify by running the test.

- [x] S3-T3.3 — Add `TestGoldensDiffer` extension (if not already done): ensure that
      `view_library_120x30.golden` (to be created in S3-T4) is included in the pairwise
      differ check, or add a separate assert that the library golden differs from the
      main-view golden at the same size.

- [x] S3-T3.4 — Add the `{"library_120x30", modeLibrary, 120, 30}` case to a
      `TestLibraryGolden` function (or extend `TestViewGolden` with a separate function)
      so that `UPDATE_GOLDEN=1` creates `testdata/view_library_120x30.golden`.
      The test model must have `m.mode = modeLibrary` and representative library content.

- [x] S3-T3.5 — Run all non-golden tests:
      `go test ./internal/ui -run "TestLibraryInMainSidebarPersists|TestLibraryViewIsTranslucent|TestNoLineExceedsWidth|TestStylesNoBackground|TestCaelestiaAccentColors"`.
      All must pass before regenerating goldens.

Estimated lines changed: ~55–80 add.

---

### S3-T4 — Create library-in-main golden; confirm no regressions

**Files**: `testdata/view_library_120x30.golden` (create),
  possibly re-regenerate existing goldens if library routing changed the main view output.
**Depends on**: S3-T3 (all non-golden tests pass)
**Spec mapping**: Golden Determinism (@slice3); Library In Main (@slice3)

- [x] S3-T4.1 — Run:
      `UPDATE_GOLDEN=1 go test ./internal/ui -run TestLibraryGolden`
      (or whatever function name was used in S3-T3.4) to create `view_library_120x30.golden`.

- [x] S3-T4.2 — Check if the main-view goldens (60×20, 80×24, 120×30, 120×40) need
      regeneration. The library routing changes only affect `modeLibrary`/`modeCreatePlaylist`;
      the default `modeNormal` path is unchanged. Run:
      `go test ./internal/ui -run TestViewGolden`
      If any fail, regenerate: `UPDATE_GOLDEN=1 go test ./internal/ui -run TestViewGolden`.

- [x] S3-T4.3 — Inspect `view_library_120x30.golden` deliberately:
      - Contains `Biblioteca` (accented nav item in sidebar).
      - Contains `Cola` (other nav items in sidebar).
      - Contains `➤` (library cursor).
      - Contains at least one tab name.
      - No line exceeds 120 columns.
      - No fully-blank body row.

- [x] S3-T4.4 — Run the full test suite:
      `go test ./internal/ui`
      All tests pass, including:
      - `TestLibraryInMainSidebarPersists` — PASS.
      - `TestLibraryGolden/library_120x30` — PASS.
      - `TestViewGolden` (all 4 sizes) — PASS.
      - `TestResultsModalGolden` — PASS (results stay full-screen, no regression).
      - `TestFooterCardNoClip60x20` — PASS (no regression).
      - `TestNoBlankBodyBand` — PASS (no regression).
      - `TestNoLineExceedsWidth` (all 4 sizes) — PASS.
      - `TestGoldensDiffer` — PASS.
      - `TestStylesNoBackground` — PASS.
      - `TestCaelestiaAccentColors` — PASS.
      - `TestDelegateNoBackground` — PASS.
      - `TestLibraryViewIsTranslucent` — PASS.
      - `TestToggleOffParity_*` — PASS.

- [x] S3-T4.5 — Run `go test ./... && go vet ./...` — green.

- [x] S3-T4.6 — Confirm scope: only `view.go`, `view_test.go`, and
      `testdata/view_library_120x30.golden` (plus possibly the 4 main goldens if they
      needed regeneration) were modified. No `styles.go`, `model.go`, `update.go`,
      `messages.go`, `keys.go`, or service file touched.

- [x] S3-T4.7 — Count changed code lines (excluding goldens): confirm < 400.

Estimated: library golden is fixture; not counted toward budget.

---

### Slice 3 — Summary table

| Task | Files | Est. lines | Design / Spec |
|------|-------|-----------|---------------|
| S3-T1 renderLibraryInMain + Biblioteca accent | `view.go` | ~60–80 add | D8, @slice3 |
| S3-T2 Route library modes into renderMain in View | `view.go` | ~25–35 mod | D8, @slice3 |
| S3-T3 TestLibraryInMainSidebarPersists + golden case | `view_test.go` | ~55–80 add | D8, @slice3 |
| S3-T4 Create library golden + confirm suite | `testdata/*.golden` | regen + 1 new | Golden Determinism |

**Total code lines**: ~140–195 add/mod (< 400 budget).

### Slice 3 — Verification checklist (apply sign-off)

- [x] `go build ./...` — green
- [x] `go vet ./...` — no findings
- [x] `go test ./internal/ui` — all pass
- [x] `go test ./...` — all packages clean
- [x] `TestLibraryInMainSidebarPersists` — PASS (sidebar persists, Biblioteca accented, ➤ in main)
- [x] `TestLibraryGolden/library_120x30` — PASS
- [x] `TestLibraryViewIsTranslucent` — PASS (no regression)
- [x] `TestResultsModalGolden` — PASS (results/pickers stay full-screen)
- [x] `TestViewGolden` (all 4 sizes) — PASS (no regression)
- [x] `TestFooterCardNoClip60x20` — PASS (no regression)
- [x] `TestNoBlankBodyBand` — PASS (no regression)
- [x] `TestNoLineExceedsWidth` (all 4 sizes) — PASS
- [x] `TestGoldensDiffer` — PASS
- [x] `TestStylesNoBackground` — PASS
- [x] `TestCaelestiaAccentColors` — PASS
- [x] `TestDelegateNoBackground` — PASS
- [x] `TestToggleOffParity_*` — PASS
- [x] modeResults / modePicker / modeLyricsPicker: still full-screen (not inside sidebar+main)
- [x] No delegate row introduces an opaque Background
- [x] No `styles.go`, `model.go`, `update.go`, `messages.go`, `keys.go` touched
- [x] Changed code lines < 400

---

## Cross-slice ordering dependencies summary

```
Slice 1 must be merged before Slice 2 begins.
Slice 2 must be merged before Slice 3 begins.

Within Slice 1:
  S1-T1 → S1-T2 → S1-T3 → S1-T4 → S1-T5 → S1-T6

Within Slice 2:
  S2-T1 → S2-T2 → S2-T3 → S2-T4 → S2-T5 → S2-T6
  (S2-T1 must precede S2-T2: new style fields used in view.go)
  (S2-T3 and S2-T4 are independent of each other but both depend on S2-T2)

Within Slice 3:
  S3-T1 → S3-T2 → S3-T3 → S3-T4
  (S3-T1 must precede S3-T2: renderLibraryInMain must exist before View routes to it)

Golden regeneration (S1-T6, S2-T6, S3-T4) always comes last in its slice,
after all non-golden tests pass. Tests land WITH their code in the same slice —
no orphaned failing tests between slices.
```
