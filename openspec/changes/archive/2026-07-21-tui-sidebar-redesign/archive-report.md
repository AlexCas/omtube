# Archive Report: tui-sidebar-redesign (Complete)

**Date:** 2026-07-21  
**Change:** TUI Sidebar Redesign — Full-Height Layout with Footer Card (3 chained slices)  
**Status:** ✅ COMPLETE

---

## Objective

Modernize and optimize the Omusic TUI layout by replacing the top-anchored single-row band of panels with a **full-height sidebar + main structure**, funded by translucent responsive foundations from `tui-visual-redesign`. The work addresses vertical wasted space on tall terminals and introduces expressive styling (accent bars, footer card, library integration):

1. **Sidebar + main structure** — Queue/nav on left sidebar (full height); artwork + lyrics in main area (right, full height).
2. **Footer card for now-playing** — Promote now-playing from top bar to a bordered, accented card between body and visualizer.
3. **Expressive styling** — Accent-bar section headers (`#e0aaff` mauve), active nav highlighting, consistent palette.
4. **Library in main** — Library renders into the main area during library mode, keeping sidebar navigation visible.
5. **Slim rail at narrow width** — Sidebar collapses to a rail below 90 columns, preserving readability.

The implementation delivers **3 chained PRs**, each independently shippable and < 400 changed lines:
- **Slice 1 (PR #26)**: Structure split (sidebar/main), no blank vertical band at 120x40. **Status**: MERGED (master).
- **Slice 2 (PR #29 consolidated)**: Expressive styling (footer card, accent bars, nav fit-gate, nav highlighting). **Status**: MERGED (master).
- **Slice 3 (PR #29 consolidated)**: Library in main, pickers/results stay full-screen. **Status**: MERGED (master).

---

## Phases Completed

| Phase | Status | Timestamp |
|-------|--------|-----------|
| explore | ✅ completed | 2026-07-21T08:20:20Z |
| propose | ✅ completed | 2026-07-21T08:24:00Z |
| spec | ✅ completed | 2026-07-21T08:24:00Z |
| design | ✅ completed | 2026-07-21T09:05:00Z |
| tasks | ✅ completed | 2026-07-21T09:12:00Z |
| apply slice 1 | ✅ completed | 2026-07-21T09:24:00Z |
| verify slice 1 | ✅ PASS-WITH-NOTES | 2026-07-21T09:33:00Z |
| judge slice 1 (retry 1) | ❌ failed — CRITICAL | 2026-07-21T15:12:46Z |
| apply slice 1 (retry 1) | ✅ completed | 2026-07-21T15:19:00Z |
| verify slice 1 (retry 1) | ✅ PASS | 2026-07-21T15:24:00Z |
| judge slice 1 (retry 1) | ❌ failed — Element Parity | 2026-07-21T15:28:37Z |
| apply slice 1 (retry 2) | ✅ completed | 2026-07-21T15:45:00Z |
| verify slice 1 (retry 2) | ✅ PASS | 2026-07-21T15:51:00Z |
| judge slice 1 (retry 2) | ✅ APROBADO | 2026-07-21T15:57:00Z |
| apply slice 2 | ✅ completed | 2026-07-21T16:20:00Z |
| verify slice 2 | ✅ PASS-WITH-NOTES | 2026-07-21T16:26:00Z |
| judge slice 2 | ✅ APROBADO | 2026-07-21T16:33:00Z |
| apply slice 3 | ✅ completed | 2026-07-21T16:45:00Z |
| verify slice 3 | ✅ PASS | 2026-07-21T16:51:00Z |
| judge slice 3 | ✅ APROBADO | 2026-07-21T16:57:00Z |
| archive | ✅ completed | 2026-07-21T17:45:00Z |

---

## Slices Summary

### Slice 1: Sidebar + Main Structure (PR #26)
**Files changed**: `styles.go` (~6 lines), `view.go` (~290 lines), `view_test.go` (~100 lines), golden testdata (4 sizes).  
**Objective**: Replace top-anchored single-row band with full-height sidebar (queue/nav) + main (artwork above lyrics). Eliminate blank vertical band at 120x40.

- Implement `sidebarW` and `mainW` computation: sidebar 30% of usable width (clamped 26–40 cols); main fills remainder.
- Slim rail below 90 cols: sidebar 22% of width (clamped 16–22 cols); main keeps max width for lyrics readability.
- **Key decision D6**: Replace `PlaceVertical(bodyH, Top, band)` with `JoinHorizontal(Top, sidebar, main)` with both children exactly `.Height(bodyH)`.
- Raise caps: `maxQueueRows = clamp(sidebarH − queueChrome, 3, sidebarH)` (from fixed 20); `lyricWindow = clamp(mainH − lyricChrome, 3, mainH)` (from fixed 12).
- Add goldens at 60×20, 80×24, 120×30, 120×40 to lock no-blank-band behavior.
- Add test assertions: `TestBodyFitsHeight()` (composed height ≤ terminal height at all sizes); `TestNoBlankBand()` (120×40 assertion); width table extended to include {120, 40}.

**Judge retry history**: 
1. **Initial judge FAILED** — 60×20 composed view rendered 22 rows in 20-row terminal (+2), clipping visualizer. Root cause: `fillBoxHeight` could not shrink content > target; lyrics sub-panel 6 inner rows vs. mainH=6 → main box grew to 8, sidebar padded to 8, view=22. Violates Element Parity @slice1 (both children == bodyH).
2. **Retry 1 judge FAILED** — Height CRITICAL resolved but element parity broken at 60×20 long queue: `queueBody` emits 6 lines (heading + ▲más + 2 rows + ▶current + ▼más) but blind tail-clip in `fillBoxHeight` dropped current `▶` + `▼ N más` marker, violating spec. Escaped verify due to short queue in tests.
3. **Retry 2 judge PASSED** — Fixed by making queue markers content-aware (preserve current `▶` + `▼ más` always, clip remaining rows if constrained); added `TestRenderQueueLongQueue()` regression asserting current track + markers survive at 60×20. Slice 1 @ commit 5196992, 399/400 non-golden lines.

**Verify result**: ✅ **PASS** (retry 2; all 40+ tests green, width assertions clean)  
**Judge result**: ✅ **APROBADO** (retry 2; dual-blind review in isolated worktrees, both prior CRITICALs resolved)

### Slice 2: Expressive Styling & Footer Card (PR #29 consolidated)
**Files changed**: `styles.go` (~50 lines), `view.go` (~150 lines), `view_test.go` (~200 lines), golden regen.  
**Objective**: Introduce expressive styling (accent bars for section headers), footer card for now-playing (replaces top bar), nav header fit-gated rendering, accent-colored active nav item.

- Define new styles: `sidebar`, `cardBorder`, `navActive`, `navItem`, `accentBar` — all NO `Background`, all using accent `#e0aaff`.
- **Footer card** design: moves now-playing from top bar to a bordered, accented card between body and visualizer; preserves full parity (▶ state, track title, progress bar, pos/dur, vol). Requires recomputing `chromeFixed`: was 11 (title + gaps + old top bar + status + help + visualizer), now 14 (title + gaps + NEW card4 + status + help + visualizer = 11 − 1_oldbar + 4_card).
- **Section headers**: queue heading, artwork heading, lyrics heading, and sidebar nav header each render with a foreground accent bar (leading ▎ glyph in `#e0aaff`).
- **Nav header fit-gate**: nav header renders only when `sidebarH >= 13` (present at 120×30/120×40, absent at 80×24/60×20) so it never displaces the mandatory queue window. Below fit-gate, `queueChrome = 5 + 0` nav rows; above, `queueChrome = 5 + 1 + navRows`.
- **Accent-colored active nav**: active nav item (queue/library/favorites/history) rendered in mauve `#e0aaff`; inactive items in muted `#a0a0a0`.
- Extend `hasNoBackground` and palette asserts to cover new sidebar/card/nav/accent-bar styles.
- Recompute `nowTitleTrunc`, `progressW` against card interior (usable − 4 for padding/border).
- Retune golden fixtures to show nav header at 120×30/120×40 but absent at 80×24/60×20.

**Design note D4 (deferred)**: Compact chrome at 60×20 (helpRows=3 instead of 2) means footer card collapses to 1 line + body separators drop, resulting in chrome=11 (same as slice 1). At taller sizes, `chromeFixed=14` standard. Reconciled in slice 1 apply retry 2.

**Deviations flagged in verify**:
1. Compact chrome: accepted by judge as minor arithmetic edge case (does not violate Element Parity; all elements fit).
2. Nav fit-gate: acceptable trade-off; queue window always preserved; nav header only cosmetic.

**Verify result**: ✅ **PASS-WITH-NOTES** (290 non-golden lines, all tests green, deviations #1/#2 acceptable)  
**Judge result**: ✅ **APROBADO** (dual-blind review, 0 blockers; nav fit-gate rule noted for final baseline spec sync)

### Slice 3: Library in Main & Finalization (PR #29 consolidated)
**Files changed**: `view.go` (~80 lines), `view_test.go` (~60 lines), golden testdata.  
**Objective**: Render library INTO the main area during library mode, keeping sidebar nav visible and accented. Keep modals (results, pickers) full-screen.

- **Library rendering path**: in `View()` when `m.mode == modeLibrary` or `modeCreatePlaylist`, render library content (tabs Playlists/Favoritos/Historial with active bracket, cursor `➤`, help) into the main area instead of full-screen; sidebar persists with nav items and queue visible; "Biblioteca" nav item accented.
- **Modal/picker preservation**: `modeResults` (search results modal), `modePicker` (list picker), `modeLyricsPicker` (lyric picker) remain full-screen (not coupled to main area). All delegate rows stay translucent (no `Background`).
- **Library help line**: synced to new footer-card-adjusted `chromeFixed`. Library mode help wrapping asserted at 60 cols.
- Add `TestLibraryRendersInMain()` asserting library tabs + cursor + help present in main area when in library mode.
- Add `TestModalsStayFullScreen()` asserting results modal and pickers render full-screen, not in main.
- Finalize palette hex asserts (mauve `#e0aaff`, teal `#00f5d4`, muted `#a0a0a0`).

**Verify result**: ✅ **PASS** (152 non-golden lines, all tests green, pickers full-screen preserved, library elements accounted for)  
**Judge result**: ✅ **APROBADO** (dual-blind review, 0 blockers; slice 3 final slice cleared)

---

## Capability Modified

| Capability | Changes |
|------------|---------|
| `caelestia-ui` | ✅ **Layout**: Full-height sidebar + main (replace top-anchored band); no blank vertical band at 120×40; slim rail below 90 cols. **Styling**: Accent bars on section headers, footer card for now-playing (replaces top bar), nav header fit-gated (present at tall sizes), active nav in accent color. **Library**: Library renders in main with persistent sidebar during library mode. **Palette**: Translucent throughout (no `Background`); accent `#e0aaff`, muted `#a0a0a0`, highlight `#00f5d4`. **Element Parity**: All existing elements preserved (queue, lyrics, artwork, now-playing, modals, pickers, library, help, visualizer). **Golden Determinism**: 60×20, 80×24, 120×30, 120×40 all differ from each other; 120×40 locks no-blank-band invariant. **Responsive**: widths/heights/caps derive from runtime terminal size; 60 cols, 80 cols, 120 cols, and 120×40 all render deterministically. Baseline spec synced and reconciled. |

---

## Build & Test Results

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ PASS (all slices, final state) |
| `go vet ./...` | ✅ PASS |
| `go test ./...` | ✅ PASS (150+ tests across 3 packages, 0 failures) |
| Slice 1 golden tests | ✅ 60×20, 80×24, 120×30, 120×40 all differ (responsive) |
| Slice 1 long-queue test | ✅ `TestRenderQueueLongQueue` PASS; current ▶ + ▼ more survive |
| Slice 1 height assert | ✅ `TestBodyFitsHeight` PASS at all 4 sizes; composed view ≤ terminal height |
| Slice 2 accent bar | ✅ New sidebar/card/nav/accent styles all NO `Background` |
| Slice 2 nav fit-gate | ✅ Nav header present at 120×30/120×40, absent at 80×24/60×20 |
| Slice 3 library assert | ✅ `TestLibraryRendersInMain` PASS; tabs + cursor + help in main area |
| Slice 3 modal assert | ✅ `TestModalsStayFullScreen` PASS; results + pickers rendered full-screen |
| Palette hex asserts | ✅ Mauve `#e0aaff`, teal `#00f5d4`, muted `#a0a0a0` all verified |
| Width assertions | ✅ No line exceeds width at 60/80/120 cols or 120×40 |
| Regression | ✅ All existing tests still pass; no `.got` residuals |

---

## Archived Files

| File | Description |
|------|-------------|
| `exploration.md` | Problem discovery: vertical wasted space (empty band at 120×30, rows 15–26), option analysis (Option A left/right stack vs. Option B sidebar/main + footer), choice of Option B. |
| `proposal.md` | Change intent: modernize layout, use vertical space, expressive styling. 3-slice strategy, capabilities affected, risks (budget per slice, visual complexity). |
| `design.md` | Technical architecture: D1 sidebar/main width formula (30% sidebar, main remainder; slim rail <90); D2 caps lifted from fixed 20/12; D3–D6 full-height join; D5 footer card chrome+3; D7 new styles (no Background); D8 library-in-main. Slices: 1 structure ~250–340; 2 expressive ~270–380; 3 library ~160–250. All <400. Tests: 120×40 golden + assertions. |
| `tasks.md` | Task breakdown for all 3 slices (S1-T1..T5, S2-T1..T4, S3-T1..T4); estimate ~900 total non-golden lines. |
| `verify-report.md` | Judge retry logs: slice 1 encountered 2 CRITICAL defects (height overflow, element parity clip) across retries 1 and 2; both resolved with targeted fixes. Slices 2 and 3 cleared on first judge pass. |
| `specs/caelestia-ui/spec.md` | Delta spec: MODIFIED requirements (Sidebar+Main, Layout Resilience, Palette+Translucency, Element Parity); ADDED (Footer Card, Accent Bars, Slim Rail, Library In Main, Golden Determinism). Slice tags @slice1/@slice2/@slice3. |
| `specs/caelestia-ui/caelestia-ui.feature` | Gherkin scenarios (full parity with spec.md requirements). |
| `SESSION_STATUS.md` | Session preflight, phase history, judge retry notes, PR stacked-PR recovery (PRs #27/#28 stacked into intermediates; #29 consolidated slices 2+3 for merge to master), baseline spec reconciliation flag. |
| `state.yaml` | Final state: phase=archive, status=completed, slices all cleared, all_phases_completed=[explore..judge], history with archive entries, artifacts updated to archive paths. |

---

## PR Merge Chain

| PR | Title | Slices | Status | Merged at |
|----|-----------|----|---------|-----------|
| #26 | Slice 1: Sidebar + main full-height layout | S1 | MERGED to master | 2026-07-21 (commit d96b625) |
| #27 | Slice 2: Expressive styling & footer card | S2 (stacked) | MERGED to S1 branch | 2026-07-21 (intermediate, not master) |
| #28 | Slice 3: Library in main | S3 (stacked) | MERGED to S2 branch | 2026-07-21 (intermediate, not master) |
| #29 | Slices 2+3 consolidated recovery (S2 @ bd0786f + S3 @ 853d728) | S2+S3 | MERGED to master | 2026-07-21 (commit 56b0917) |

**Stacked-PR recovery context**: Initial chain #26→#27→#28 routed to intermediate branches (S2 → S1 branch, S3 → S2 branch). To land slices 2+3 on master, a consolidated PR #29 was created merging both slices directly to master without passing through master's #26 commits (recovery maneuver to avoid intermediate branch divergence). All 3 slices now live on master; full redesign complete.

**Commit authorship**: User's sole git authorship retained; no Co-Authored-By trailers in any commit message per project policy.

---

## Baseline Spec Synchronization

### Prior Gap & Reconciliation

The baseline `openspec/specs/caelestia-ui/spec.md` was STALE — it pre-dated `tui-visual-redesign` and still showed:
- Opaque primary `#1a1a2e` as "primary (deep blue)" in the palette table
- No responsive/translucency requirements (those were added by `tui-visual-redesign` but never synced back to baseline)

**Resolution**: When this change archives, the baseline has been synced to reflect:
1. **Palette clarification**: Replaced "primary (deep blue) `#1a1a2e`" entry with accurate representation: translucent system (no opaque Background fill); working palette is accent `#e0aaff`, muted `#a0a0a0`, highlight `#00f5d4` with no primary background color.
2. **Requirements reconciliation**: Merged responsive requirements from `tui-visual-redesign` (widths/heights derive from runtime, 60/80/120 breakpoints) + new sidebar/main/footer/accent/library requirements from this change. The baseline now coherently describes the post-both-changes state.

---

## Design Decisions & Tradeoffs Finalized

| Decision | Rationale | Status |
|----------|-----------|--------|
| **D1: Sidebar 30% / main 70%** | Balances nav visibility with lyrics readability; slim rail <90 cols (22%) maximizes lyrics width at narrow sizes | ✅ Locked by slice 1 |
| **D2: Caps lifted from fixed 20/12** | Use available height rather than hardcoded ceilings; footprint grows gracefully to 120×40 without blank band | ✅ Locked by slice 1 |
| **D3–D6: Full-height join (no PlaceVertical top-anchor)** | Eliminates empty vertical band; both children exactly bodyH; responsive to height | ✅ Locked by slice 1; fixed via element-parity retry 2 |
| **D5: Footer card (chromeFixed +3)** | Modernizes now-playing presentation; reclaims space from removed top bar; fits slice 2 within 400-line budget | ✅ Locked by slice 2 |
| **D7: Accent bars (foreground glyphs, no Background)** | Preserves translucency invariant; establishes visual hierarchy; palette-driven | ✅ Locked by slice 2 |
| **D7c: Nav header fit-gated at sidebarH>=13** | Trade-off: nav header cosmetic; queue window mandatory; renders at 120×30/40, absent at 80×24/60×20 | ✅ Locked by slice 2 design; noted for future baseline |
| **D8: Library in main (modals/pickers full-screen)** | Maintains mode clarity; library tabs/cursor/help visible during search/edit; modals never obscure sidebar context | ✅ Locked by slice 3 |

---

## Known Upstream Deferred Debt

None carried forward from prior `tui-visual-redesign` change. New debt identified: Below 60×18/19 rows (out of spec floor), queue clip reappears — consider a min-size guard in a future optimization slice (not critical for 60×20 spec floor).

---

## Final Status

✅ **All 3 slices merged to master, all judge retries cleared, and archive complete.**

- Omusic TUI now uses full vertical space on tall terminals (120×40 shows no blank band).
- Layout responsive across width (60/80/120 cols) and height (raises caps to bodyH-derived); sidebar full-height + main full-height via join.
- Expressive styling: accent bars, footer card, nav highlighting, consistent palette (`#e0aaff` accent, `#a0a0a0` muted, `#00f5d4` highlight).
- Library mode integrated into main area; sidebar navigation remains visible and accented; modals (results, pickers) stay full-screen.
- All existing elements preserved: queue, lyrics, artwork (hidden <90 cols), now-playing, title, help, visualizer, modals, pickers, library.
- Build, vet, and test suites fully green (150+ tests, 0 failures).
- Golden fixtures locked at 4 sizes (60×20, 80×24, 120×30, 120×40), proving deterministic responsiveness.
- Baseline `caelestia-ui` spec reconciled and synced: translucency palette accurate, responsive/sidebar/footer/library/accent requirements captured.

**Change successfully archived and merged to master.**
