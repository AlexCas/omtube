# Tasks — TUI Visual Redesign: Slice 1 (Base)

Scope: purely presentational changes to `internal/ui/styles.go`, `internal/ui/view.go`,
and `internal/ui/view_test.go`. No Model/Update/messages/keys/services touched.
Estimated changed lines: ~180–240 (well under 400-line budget).

---

## Order of execution

Tasks MUST be executed in numbered order. Dependencies are noted inline.

---

## T1 — Remove opaque backgrounds from styles.go

**File**: `internal/ui/styles.go`
**Depends on**: nothing (first change)
**Spec mapping**: @slice1 "No opaque background paints over the terminal glass"
  (spec.md: "Caelestia Palette", feature: "No opaque background paints over the terminal glass")

- [ ] T1.1 — Remove `.Background(lipgloss.Color("#1a1a2e"))` from the `title` style
      (currently line ~21). Keep `.Bold(true)`, `.Foreground(#e0aaff)`,
      `.Border(lipgloss.RoundedBorder())`, `.BorderForeground(#e0aaff)`, `.Padding(0,1)`.
- [ ] T1.2 — Remove `.Background(lipgloss.Color("#1a1a2e"))` from the `panel` style
      (currently line ~26). Keep `.Border(lipgloss.RoundedBorder())`,
      `.BorderForeground(#e0aaff)`, `.Padding(0,1)`.
- [ ] T1.3 — Verify `go build ./...` passes with no errors after this change.

Estimated lines changed: ~4 deletions.

---

## T2 — Add layout types and computeLayout to view.go

**File**: `internal/ui/view.go`
**Depends on**: T1 (build must be green before adding new code)
**Spec mapping**: @slice1 "Widths derive from runtime dimensions", "No rendered line exceeds terminal width"
  (design.md: Decision 1 — Breakpoint thresholds; Decision 3 — Fluid width formula)

- [ ] T2.1 — Add `breakpoint` type and constants at the top of `view.go` (after imports):
      ```go
      type breakpoint int
      const (
          bpNarrow breakpoint = iota // < 90 cols
          bpMedium                   // 90–119 cols
          bpWide                     // >= 120 cols
      )
      ```
- [ ] T2.2 — Add `classify(width int) breakpoint` function:
      - `width < 90`  → `bpNarrow`
      - `width < 120` → `bpMedium`
      - else          → `bpWide`
- [ ] T2.3 — Add `layout` struct with fields:
      `bp`, `queueW`, `lyricsW`, `artW`, `progressW`,
      `maxQueueRows`, `lyricWindow`, `plainLines`,
      `nowTitleTrunc`, `libLineTrunc`, `showArtwork int/bool`
      (exact types per design.md Decision 3).
- [ ] T2.4 — Add `computeLayout(width, height int) layout` function:
      - `usable = max(width-2, minUsable)` (define `minUsable = 40`)
      - Per breakpoint, compute `queueW`, `lyricsW`, `artW` as `round(usable * pct)`,
        clamped (queue min 24 / lyrics min 28 / artwork fixed 24–28).
      - Fold remainder into `lyricsW` so `queueW + lyricsW + artW == usable`
        (and `artW == 0` for `bpNarrow`).
      - `showArtwork = bp != bpNarrow`
      - `progressW = clamp(width-24, 8, 40)` (decorLen ~24: state+times+vol chrome).
      - `nowTitleTrunc = max(8, lyricsW-4)` (reasonable title trunc, revised at verify).
      - `maxQueueRows = 10` (Slice 1 keeps current value; dynamic from height is Slice 2).
      - `lyricWindow = 7` (Slice 1 keeps current value; dynamic from height is Slice 2).
      - `plainLines = 8` (Slice 1 keeps current value).
      - `libLineTrunc = max(20, width-4)`.
- [ ] T2.5 — Add unexported helper `clamp(v, lo, hi int) int` if not already present.
- [ ] T2.6 — Verify `go build ./...` passes after T2.

Estimated lines added: ~60–75.

---

## T3 — Apply fluid widths and truncations in view.go render helpers

**File**: `internal/ui/view.go`
**Depends on**: T2 (layout types must exist)
**Spec mapping**: @slice1 "Widths derive from runtime dimensions", "No rendered line exceeds terminal width"
  (design.md: File Changes — view.go; Decision 3)

- [ ] T3.1 — In `View()`, call `l := computeLayout(m.width, m.height)` at the top
      (after the early-return guards, before any render calls). Thread `l` into each
      render helper call.
- [ ] T3.2 — Change `renderQueue()` signature to `renderQueue(l layout) string`.
      Replace hardcoded `Width(36)` → `m.styles.panel.Width(l.queueW)`.
      Replace hardcoded truncation `28` → `l.queueW - 2`.
      Replace constant `maxQueueRows` → `l.maxQueueRows`.
- [ ] T3.3 — Change `renderNowPlaying()` signature to `renderNowPlaying(l layout) string`.
      Replace hardcoded `progressBar(..., 30)` → `progressBar(m.pos, m.dur, l.progressW)`.
      Replace hardcoded title truncation `32` → `l.nowTitleTrunc`.
- [ ] T3.4 — Change `renderLyricsPanel()` signature to `renderLyricsPanel(l layout) string`.
      Replace hardcoded `Width(50)` → `m.styles.panel.Width(l.lyricsW)`.
      Replace hardcoded plain-lyrics args `(48, 8)` → `(l.lyricsW-2, l.plainLines)`.
      Pass `l` into `renderSyncedLyrics(l layout)`.
- [ ] T3.5 — Change `renderSyncedLyrics()` signature to `renderSyncedLyrics(l layout) string`.
      Replace hardcoded `window = 7` → `l.lyricWindow`.
      Replace hardcoded truncation `46` → `l.lyricsW - 4`
      (extra 2 for the "▶ " prefix so the line stays within the inner width).
- [ ] T3.6 — Change `renderArtworkPanel()` signature to `renderArtworkPanel(l layout) string`.
      Replace hardcoded `Width(28)` → `m.styles.panel.Width(l.artW)`.
- [ ] T3.7 — Change `renderEnrichment()` signature to `renderEnrichment(l layout) string`.
      Pass `l` to `renderLyricsPanel(l)` and `renderArtworkPanel(l)`.
      NOTE: Slice 1 does NOT hide artwork for `bpNarrow` — that is Slice 2.
      Slice 1 only ensures widths are fluid so no overflow at 60/80/120.
- [ ] T3.8 — Change `renderMiddleSection()` signature to `renderMiddleSection(l layout) string`.
      Pass `l` to `renderQueue(l)` and `renderEnrichment(l)`.
- [ ] T3.9 — In `trackLines()` (renderLibList helper), replace hardcoded truncation `60`
      with `l.libLineTrunc` — requires passing `l` into `renderLibrary()` and
      `renderLibList()` as well, OR extracting the trunc constant from `trackLines`
      by making it accept a `maxCols int` parameter and calling it as
      `trackLines(m.libFavorites, l.libLineTrunc)`.
      Prefer the simpler second option to avoid cascading signature changes in library code.
- [ ] T3.10 — Verify `go build ./...` passes after T3.

Estimated lines changed: ~60–80 (signature changes + replacements).

---

## T4 — Add test assertions in view_test.go (BEFORE regenerating goldens)

**File**: `internal/ui/view_test.go`
**Depends on**: T3 (code must compile; asserts must pass against new behavior)
**Spec mapping**: @slice1 "No rendered line exceeds terminal width", "No opaque background paints over the terminal glass", "80x24 and 120x30 goldens differ"
  (design.md: Decision 2 — no-Background assert; Decision 5 — golden test strategy)

- [ ] T4.1 — Add `hasNoBackground(s lipgloss.Style) bool` helper in `view_test.go`:
      returns `s.GetBackground() == lipgloss.Color("")`.
- [ ] T4.2 — Add `TestStylesNoBackground(t *testing.T)`:
      constructs `defaultStyles()`, asserts `hasNoBackground(s.title)` and
      `hasNoBackground(s.panel)`. Maps to spec scenario "No opaque background paints
      over the terminal glass".
- [ ] T4.3 — Add `TestNoLineExceedsWidth(t *testing.T)`:
      table test over widths `[]int{60, 80, 120}`. For each:
      - create a test model with that width and a representative height (24).
      - call `m.View()`, split on `\n`.
      - assert `lipgloss.Width(line) <= width` for every non-empty line.
      Maps to spec scenario "No rendered line exceeds terminal width" (all three Examples).
- [ ] T4.4 — Add `TestGoldensDiffer(t *testing.T)`:
      reads `testdata/view_80x24.golden` and `testdata/view_120x30.golden` as bytes.
      Asserts `!bytes.Equal(want80, want120)`.
      NOTE: this test will fail until golden files are regenerated in T5. Mark with
      `t.Skip("run after UPDATE_GOLDEN=1")` initially, or add a guard:
      if either file is missing, `t.Skip(...)`.
      Maps to spec scenario "80x24 and 120x30 goldens differ".
- [ ] T4.5 — Run `go test ./internal/ui/... -run TestStylesNoBackground` — must pass.
- [ ] T4.6 — Run `go test ./internal/ui/... -run TestNoLineExceedsWidth` — must pass.
- [ ] T4.7 — Run `go test ./internal/ui/... -run TestViewGolden` — expected FAIL
      (goldens are stale; confirms the code changed). Note the failure is expected at
      this step.

Estimated lines added: ~55–70.

---

## T5 — Regenerate golden fixtures and verify full test suite

**Files**: `internal/ui/testdata/view_80x24.golden`, `internal/ui/testdata/view_120x30.golden`
**Depends on**: T4 (all non-golden asserts must pass first; goldens must be regenerated
against the new layout code)
**Spec mapping**: @slice1 "80x24 and 120x30 goldens differ", "Golden Determinism"
  (design.md: Decision 5 — golden test strategy; spec.md "Golden Determinism")

- [ ] T5.1 — Run `UPDATE_GOLDEN=1 go test ./internal/ui/... -run TestViewGolden`
      to regenerate `view_80x24.golden` and `view_120x30.golden`.
- [ ] T5.2 — Inspect the diff of both golden files:
      - Confirm `view_80x24.golden` uses narrower panel widths (bpNarrow or bpMedium).
      - Confirm `view_120x30.golden` uses wider panel widths (bpWide).
      - Confirm NO line in either file is visually wider than its target width.
      - Confirm the two files differ in content (different widths, different column counts).
- [ ] T5.3 — Remove the `t.Skip` guard from `TestGoldensDiffer` (added in T4.4).
- [ ] T5.4 — Run `go test ./internal/ui/...` (full suite, no UPDATE_GOLDEN):
      - `TestStylesNoBackground` — PASS.
      - `TestNoLineExceedsWidth` — PASS.
      - `TestGoldensDiffer` — PASS.
      - `TestViewGolden` (80x24, 120x30) — PASS.
      - `TestToggleOffParity_*` (existing parity tests) — PASS.
- [ ] T5.5 — Run `go build ./...` one final time to confirm the full module builds clean.

Estimated lines changed: golden files regenerated (not counted toward code budget).

---

## Summary table

| Task | Files | Est. lines | Spec/Scenario |
|------|-------|-----------|---------------|
| T1 Remove opaque bg | `styles.go` | ~4 del | Caelestia Palette, No opaque bg |
| T2 Layout types + computeLayout | `view.go` | ~65 add | Widths derive from runtime |
| T3 Fluid widths in render helpers | `view.go` | ~70 mod | No line exceeds width, Widths derive |
| T4 Test assertions | `view_test.go` | ~65 add | No bg assert, Width assert, Goldens differ |
| T5 Regen goldens + full suite | `testdata/*.golden` | regen | Golden Determinism, 80x24 != 120x30 |

**Total code lines (styles + view + test)**: ~200 lines changed/added (< 400 budget).
Golden files are regenerated, not authored — not counted toward budget.

---

## Verification checklist (for apply phase sign-off)

- [ ] `go build ./...` — green
- [ ] `go test ./internal/ui/...` — all tests pass
- [ ] `TestStylesNoBackground` — passes (no Background on title/panel)
- [ ] `TestNoLineExceedsWidth` at 60, 80, 120 — passes
- [ ] `TestGoldensDiffer` — passes (80x24 != 120x30)
- [ ] `TestViewGolden` 80x24 and 120x30 — pass (goldens match regenerated output)
- [ ] No Model/Update/messages/keys/services files touched
- [ ] No Slice 2 or Slice 3 code introduced
