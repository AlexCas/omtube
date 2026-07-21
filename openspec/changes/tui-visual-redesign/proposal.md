# Proposal: TUI Visual Redesign — Responsive Dashboard on Glass

## Intent

Two defects break the Omusic TUI: opaque `Background(#1a1a2e)` fills
(`styles.go:21,26`) paint over glass/blur terminals, and hardcoded panel
widths/windows ignore `m.width`/`m.height`, overflowing at 80 cols and wasting
space at 120. Goal: a responsive dashboard (exploration Option C) that uses
terminal HEIGHT, respects translucency, keeps mauve/teal accents, and preserves
every existing element.

## Scope

### In Scope
- Remove the two opaque `Background` fills → transparent by default (keep rounded borders).
- Responsive layout: narrow/medium/wide breakpoints derived from `m.width`/`m.height`; vertical fill.
- Fluid widths/truncations/windows from runtime dimensions.
- Hide artwork panel below a width breakpoint (do not shrink or move).
- Restyle modals/pickers (`modeResults`, library, lyric pickers) for coherence.
- Regenerate goldens + add a 60×20 case, a `lipgloss.Width(line) <= width` assert, and a no-`Background` assert.

### Out of Scope
- No changes to `Model`, `Update`, `messages`, `keys`, or services.
- No keybinding or behavior changes.
- No configurable themes / new palette.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `caelestia-ui`: styles MUST be transparent (no opaque `Background`); layout MUST be responsive across width AND height with narrow/medium/wide breakpoints; artwork MUST hide below the width breakpoint; no rendered line may exceed terminal width.

## Approach

Purely presentational; confined to `styles.go`, `view.go`, and test fixtures.
A layout model computes a breakpoint from `m.width` (narrow < ~80, medium ~80–120,
wide ≥ ~120) and available height, then derives panel widths, progress-bar width,
truncations, `maxQueueRows`, and lyric windows via `lipgloss.Width/JoinHorizontal/
JoinVertical/Place`. Delivered as 3 chained slices, each < 400 changed lines:

| Slice | Scope | Files | Est. lines | Deliverable at end |
|-------|-------|-------|-----------|--------------------|
| 1 Base | Remove 2 `Background` fills; fluid widths/truncations/windows from `m.width/m.height`; narrow breakpoint stops 80-col overflow; keep current shape | `styles.go`, `view.go`, goldens, `view_test.go` | ~150–220 | Glass fixed, no overflow; goldens differ per size; width assert + no-bg assert pass |
| 2 Dashboard | Proportional regions using height (`Place`/`PlaceVertical`); narrow/medium/wide columns; hide artwork below breakpoint | `view.go`, goldens, `view_test.go` | ~200–320 | Height used; 60×20 case locked; three breakpoints render distinctly |
| 3 Modals | Restyle `modeResults`, library, lyric pickers (themed `list` delegate w/o opaque row bg) | `view.go`, `styles.go`, goldens | ~150–250 | Modals/pickers coherent; still transparent |

## Preserved Elements (parity checklist)

- [ ] Title "🎵 Omusic" / "📚 Biblioteca"
- [ ] Now-playing bar: ▶/⏸ state, title, progress, time, volume
- [ ] Shared search/prompt/input
- [ ] Status line
- [ ] Queue panel: sliding window, "▲/▼ N más", cache mark ⤓, current ▶
- [ ] Lyrics panel: synced highlight + plain fallback
- [ ] Artwork panel (hidden only below narrow breakpoint)
- [ ] Help line
- [ ] Bar visualizer
- [ ] Library mode: tabs Playlists/Favoritos/Historial, cursor, create playlist
- [ ] Results modal + pickers

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/ui/styles.go` | Modified | Drop opaque backgrounds; keep borders/accents |
| `internal/ui/view.go` | Modified | Responsive layout math, breakpoints, vertical fill, modal styling |
| `internal/ui/testdata/*.golden` | Modified | Regenerated per-size; new 60×20 fixture |
| `internal/ui/view_test.go` | Modified | Width assert, no-`Background` assert, narrow case |

## Verification Strategy

- Regenerate goldens (`UPDATE_GOLDEN=1 go test ./internal/ui`); assert 80×24 and 120×30 fixtures now DIFFER (proves responsiveness).
- Add 60×20 golden to lock the narrow single-column breakpoint.
- Assert `lipgloss.Width(line) <= width` for every rendered line (guards overflow).
- Assert `title`/`panel` styles expose no `Background` (guards translucency).
- TUI in Go — no Playwright.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Golden churn masks a real regression | Med | Regenerate deliberately per slice; rely on width/no-bg asserts, not only byte goldens |
| Cross-size layout regressions | Med | Explicit 60×20/80×24/120×30 cases; assert fixtures differ |
| `bubbles/list` delegate re-introduces opaque row bg | Med | Slice 3 themed delegate uses foreground/border only; no-bg assert extended to modal styles |
| A slice exceeds 400-line budget | Low | Split enforced per table; re-slice if forecast crosses budget at tasks phase |

## Rollback Plan

Each slice is an isolated PR touching only presentation files. Revert the slice's
commit; earlier slices remain valid since none touch `Model`/`Update`/services.
No data or config migration involved.

## Dependencies

- Existing `bubbletea v1.3.10`, `lipgloss v1.1.0`, `bubbles v1.0.0` (no new deps).

## Success Criteria

- [ ] No opaque `Background` on `title`/`panel`; glass shows through (no-bg assert passes).
- [ ] No rendered line exceeds terminal width at 60×20, 80×24, 120×30.
- [ ] 80×24 and 120×30 goldens differ; height is used for region sizing.
- [ ] Artwork hidden below the width breakpoint; all other elements preserved.
- [ ] Modals/pickers visually coherent and transparent.
- [ ] Each PR < 400 changed lines.
