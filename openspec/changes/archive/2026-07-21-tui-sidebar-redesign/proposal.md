# Proposal: TUI Sidebar Redesign — Full-Height Sidebar + Main Content on Glass

## Intent

The prior change `tui-visual-redesign` misread "preserve elements" as "preserve the
layout", shipping only a cosmetic pass. The layout is still top-anchored (title →
now-playing → status → one loose panel band → help → visualizer), leaving a wasted
vertical band (twelve blank rows at 120×30, `view.go:527` `PlaceVertical(bodyH, Top,
band)` + the 12/20 window caps) and a flat, hierarchy-free look. This change delivers
the redesign the user actually wanted — a substantially different **layout and
aesthetic** — while inventing NO new functionality. Purely presentational: a
full-height left **sidebar** (accent-bar nav + queue) joined to a **main content
area** (artwork + lyrics) via `JoinHorizontal`, both sized to `bodyH` so the blank
band disappears by construction; now-playing promoted to a **footer card**; a
**more expressive** pass over the existing Caelestia mauve/teal palette.

## Scope

### In Scope
- Sidebar + main split in `computeLayout` (`sidebarW`/`mainW`/`sidebarH`/`mainH`);
  replace top-anchored band with `JoinHorizontal(Top, sidebar, main)`, both `bodyH` tall.
- Raise/derive the `maxQueueRows` (20) and `lyricWindow`/`plainLines` (12) caps from
  `bodyH` so content fills the vertical space.
- Accent-bar section headers; static nav list (Cola/Biblioteca/Favoritos/Historial,
  active accented); now-playing footer card (re-measure `chromeFixed`); richer state
  colors; artwork stacked above lyrics in main.
- Narrow (<90): sidebar collapses to a **slim rail** (lyrics keep max width) — no
  stacking, no full hide.
- Render `renderLibrary` INTO the main area with a persistent sidebar.
- Regenerate goldens (60×20, 80×24, 120×30) and ADD a tall **120×40** case; extend
  `hasNoBackground` asserts to every new style (including the `bubbles/list` delegate).

### Out of Scope
- No changes to `Model`, `Update`, `messages`, `keys`, or services.
- No new keybindings or behavior; no sidebar-focus navigation.
- `modeResults`/pickers stay full-screen (`bubbles/list` in `Update`) to avoid coupling.
- No configurable themes / new palette.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `caelestia-ui`: layout MUST be a full-height sidebar + main content area
  (`JoinHorizontal`, both `bodyH` tall — no reappearing blank band); content windows
  MUST derive from `bodyH` (not fixed 12/20 caps); now-playing MUST render as a footer
  card; narrow width MUST collapse the sidebar to a slim rail (not stack/hide); library
  MUST render in the main area with a persistent sidebar; NO opaque `Background` on any
  style, delegate row, or list title; no rendered line may exceed terminal width.

## Approach

Presentational only, confined to `internal/ui/styles.go`, `internal/ui/view.go`,
`internal/ui/view_test.go`, `internal/ui/testdata/*.golden`. Exploration Option B.
`computeLayout` gains a top-level `sidebarW`/`mainW` split (sidebar ~28–34% of
`usable`, clamped; main takes remainder; slim rail at narrow), plus `sidebarH`/`mainH`
= `bodyH`. Renderers build full-height bordered boxes; caps derive from those heights.
Delivered as 3 chained PRs, each < 400 changed lines (force-chained, 400-line budget):

| Slice | Scope | Files | Est. lines | Deliverable |
|-------|-------|-------|-----------|-------------|
| **1 Structure** | Sidebar+main split in `computeLayout`; replace top-anchored band with `JoinHorizontal(Top, sidebar, main)` both `bodyH` tall; derive caps from `bodyH`; slim-rail narrow; now-playing stays a top bar (intermediate). Retune width/height asserts; regenerate 3 goldens + add 120×40. | `view.go`, `view_test.go`, goldens | ~230–330 | Blank band GONE; sidebar/main fill `bodyH`; invariants pass |
| **2 Expressive styling** | `sectionHeader` accent bars; sidebar nav header (active accented); now-playing → footer **card** (re-measure `chromeFixed`); richer state colors; artwork with presence. New styles no-`Background`; extend `hasNoBackground` asserts. Regenerate goldens. | `styles.go`, `view.go`, `view_test.go`, goldens | ~260–370 | Footer card + nav header; translucency asserted on new styles |
| **3 Library in main** | Render `renderLibrary` INTO main (sidebar persists, "Biblioteca" accented); pickers/results stay full-screen. Add library-in-main golden + sidebar-persists assert. Regenerate goldens. | `view.go`, `view_test.go`, goldens | ~160–250 | Library in main with persistent sidebar |

Re-slice forecast: if Slice 2 crosses 400 at tasks, split into 2a (accent headers +
nav) and 2b (footer card + chrome re-measure). No current forecast crosses 400.

## Preserved Elements (parity checklist)

- [ ] Title "🎵 Omusic" (main) / "📚 Biblioteca" (library)
- [ ] Now-playing: ▶/⏸ state glyph, title, progress bar, `pos/dur` time, `vol N`
- [ ] Shared search/prompt input (search, URL, import URL/name, lyrics search)
- [ ] Status line (`m.status`)
- [ ] Queue panel: `Cola (N)`, sliding window, `▲/▼ N más`, cache mark `⤓`, current `▶`
- [ ] Lyrics panel: synced `▶` highlight; plain fallback; `sin letra` empty state
- [ ] Artwork panel: art or `[sin portada]`; hidden below width breakpoint
- [ ] Help line (wrapped to width)
- [ ] Bar visualizer (animated while playing, flat otherwise)
- [ ] Library: tabs Playlists/Favoritos/Historial, cursor `➤`, create-playlist prompt, help
- [ ] Results modal + pickers (`modePicker`, `modeLyricsPicker`)
- [ ] Translucency: NO opaque `Background` on any style, delegate row, or list title
- [ ] Caelestia palette (`#e0aaff` / `#00f5d4` / `#a0a0a0`)
- [ ] No rendered line exceeds terminal width at 60/80/120

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/ui/view.go` | Modified | Sidebar+main layout math, `JoinHorizontal`, `bodyH`-derived caps, footer card, library-in-main, slim rail |
| `internal/ui/styles.go` | Modified | Accent-bar headers, nav-active/card styles (all no-`Background`) |
| `internal/ui/testdata/*.golden` | Modified | Regenerated per size; new 120×40 fixture; library-in-main fixture |
| `internal/ui/view_test.go` | Modified | Retuned width/height asserts; extended `hasNoBackground`; 120×40 + sidebar-persists asserts |

## Verification Strategy

- Regenerate goldens (`UPDATE_GOLDEN=1 go test ./internal/ui`); review each `.got` diff.
- Add 120×40 golden; assert no blank vertical band (sidebar/main both fill `bodyH`).
- Assert `lipgloss.Width(line) <= width` for every line at 60×20/80×24/120×30/120×40.
- Assert no `Background` on all new styles + delegate + list title (extend asserts).
- Retune/keep `TestClassifyBoundaries`, `TestComputeLayoutWidths/Height`,
  `TestCaelestiaAccentColors`, toggle-off parity tests. Go TUI — no Playwright.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Golden churn masks a regression | High | Regenerate per slice; lean on width/no-bg/palette asserts, not only bytes; review each `.got` |
| Delegate/library-in-main re-adds opaque row bg | Med | Foreground/border-only; extend `TestDelegateNoBackground`/`TestLibraryViewIsTranslucent`; keep pickers on `bubbles/list` |
| Breakpoint edges (89/90, 119/120) break the split | Med | Retune `TestClassifyBoundaries`/`TestComputeLayoutWidths`; assert `sidebarW+mainW <= usable` at boundaries |
| Footer-card chrome re-measure clips elements at 20 rows | Med | Keep `minBody` floor + row mins; 60×20 asserts title, now-playing, queue+≥3 rows, help, visualizer present |
| Slice 2 exceeds 400 lines | Med | Pre-planned 2a/2b split; enforce at tasks |
| Slim rail crowds lyrics at narrow | Med | Lyrics keep max width; assert no line exceeds width at 60/80 and lyrics present |
| "More expressive" over/undershoots (subjective) | Low | Lock concrete testable decisions in design; goldens pin exact output |

## Rollback Plan

Each slice is an isolated PR touching only presentation files. Revert the slice's
commit; earlier slices remain valid since none touch `Model`/`Update`/services. No
data or config migration involved.

## Dependencies

- Existing `bubbletea v1.3.10`, `lipgloss v1.1.0`, `bubbles v1.0.0` (no new deps).

## Success Criteria

- [ ] No reappearing blank vertical band at 120×40; sidebar + main both fill `bodyH`.
- [ ] Now-playing footer card present.
- [ ] No opaque `Background` on any new style (assert passes), including the
      `bubbles/list` delegate.
- [ ] Every rendered line ≤ terminal width at 60×20, 80×24, 120×30, 120×40.
- [ ] All prior elements preserved (parity checklist passes).
- [ ] Each PR < 400 changed lines.
