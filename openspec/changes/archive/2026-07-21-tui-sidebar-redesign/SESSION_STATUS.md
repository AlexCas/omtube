# SESSION STATUS

**Active change**: `tui-sidebar-redesign`
**Current phase**: judge complete — 3 chained PRs OPEN; archive DEFERRED until merge
**Status**: whole change judge-PASS, visually verified live. Artifacts committed on slice1 (d96b625). Chain rebased & pushed.
**Updated**: 2026-07-21T17:05:00Z

## PR status (stacked-PR mishap + recovery)
- #26 slice 1 → master: MERGED ✓ (master has slice 1)
- #27 slice 2 → slice1 branch: MERGED into intermediate branch, NOT master ✗
- #28 slice 3 → slice2 branch: MERGED into intermediate branch, NOT master ✗
- **#29 slice3 → master: OPEN** — consolidated recovery PR bringing slice 2 (bd0786f) + slice 3 (853d728) to master. mergeable=MERGEABLE, state=BLOCKED (branch protection awaiting approval).

**master currently = slice 1 only.** Full redesign lands when #29 merges.

## Live visual verification (done)
Built /tmp/omusic-tui, ran under tmux 120x40. Normal mode: full-height sidebar (nav + queue) + main (Portada/Letra), footer card, NO empty band. Library mode (L): sidebar persists with ▸ Biblioteca accented, library renders in main. Matches the redesign intent.

## Remaining: archive (after PRs merge)
When #26/#27/#28 merge to master, run archive: move openspec/changes/tui-sidebar-redesign → openspec/changes/archive/2026-07-21-tui-sidebar-redesign, sync the caelestia-ui baseline spec (also reconcile the pre-existing stale baseline — option A deferral from spec phase), and MOVE this SESSION_STATUS.md into the archived folder. Prior change was archived in a separate post-merge commit — replicate.
**PR decision**: keep all 3 slices on local chained branches; push + open 3 chained PRs at the very end.

## Slice 2 apply — deviations to scrutinize in verify/judge
1. Compact chrome at tight heights: helpRows(60)=3 (design assumed 2). At 60x20 the footer card collapses to a single line + body separators drop → chrome 11, bodyH=6, view exactly 20 rows. chromeFixed=14 stays the standard measure for taller sizes.
2. Nav header fit-gated: renders only when sidebarH>=13 (appears at 120x30/120x40, absent at 80x24/60x20) so it never displaces the mandatory queue window. queueChrome = 5+navRows.
3. Accent bars as leading ▎ glyph (D7c-sanctioned) not underline rule.
4. nowTitleTrunc/progressW re-derived vs card interior (usable-4).
5. TestComputeLayoutHeight retuned to derived-from-height pins (raw 20/12 thresholds now unreachable with nav+card).
## Slice status
- slice 1: CLEARED (feat/tui-sidebar-redesign-slice1 @ 5196992)
- slice 2: CLEARED (feat/tui-sidebar-redesign-slice2 @ 6efcc02)
- slice 3: CLEARED (feat/tui-sidebar-redesign-slice3 @ 223f2a1)
**Budget watch**: slice 1 at 399/400 non-golden lines — at the ceiling (no more headroom in this slice).
**INFO (deferred)**: below documented floor (60x18/19, out of spec) the queue clip reappears — consider a min-size guard in a later slice.

## Slice status
- slice 1 (Structure): CLEARED — feat/tui-sidebar-redesign-slice1 @ 5196992, judge PASS
- slice 2 (Expressive styling): pending
- slice 3 (Library in main): pending

## Next recommended step
Confirm with user: push slice-1 branch + open chained PR now (then branch slice 2 off it), vs continue applying slice 2 locally first. Then apply→verify→judge slice 2.

## Judge slice 1 retry 1 — new defect (retry 2 of 3)
Height overflow RESOLVED. But `fillBoxHeight`'s blind tail-clip (`lines[:inner]`, view.go:571) drops MANDATORY content at 60x20 + long queue: `queueBody` emits 6 lines (heading + ▲más + 2 rows + ▶current + ▼más) but inner=4 → current `▶` track + `▼ N más` clipped. Violates Element Parity @slice1 (spec:122 "current ▶"). Design line 379 sanctioned an ORDERED content-aware clip, not blind truncation. Escaped verify because all visual tests use a 2-item queue.
**Fix**: root-cause in computeLayout (view.go:147-148) — `maxQueueRows`/`queueChrome` must reserve room for heading + BOTH scroll markers so `heading+markers+window <= sidebarH-panelBorder` at the floor (or drop markers when window is at floor); never let queueBody exceed inner. If keeping fillBoxHeight as safety net, clip parity-preserving (never drop current ▶, keep >=3 queue rows). Add long-queue (n>=20) 60x20 regression test asserting current track + ▶ present.

## Judge slice 1 — CRITICAL (retry 1 of 3)
At 60x20 the composed `View()` renders 22 rows in a 20-row terminal (+2 overflow), clipping the visualizer (mandatory element). Root cause: `fillBoxHeight` (view.go:563-569) uses `.Height(rows-panelBorder)` but lipgloss `.Height()` cannot shrink content already taller than target; the intermediate side-by-side lyrics sub-panel needs 6 inner rows but mainH=6 gives 4 → main box grows to 8, sidebar padded to 8. Violates @slice1 "both children exactly bodyH" (D2a/D6). The earlier verify deviation #3 pre-flagged this; the missing invariant assert was the gap.
**Fix**: clamp lyricWindow/plainLines against the sub-panel's own chrome so it fits mainH-panelBorder at small heights (or clip); add `TestBodyFitsHeight` asserting `lipgloss.Height(m.View()) <= m.height` at 60x20/80x24/120x30/120x40; regenerate view_60x20.golden.
**Chained-PR note**: apply/verify/judge slice-by-slice; no push/PR without explicit user confirmation.
**Slice 1 apply**: 319 non-golden lines (<400). go test/vet green. Blank band eliminated at 120x40. 6 arithmetic deviations from design.md logged (lyricChrome=5, lyricsW/artW fit inside mainW, .Height(bodyH-border), height-30 assert >=12, nowTitleTrunc width-bound, update_test.go cap test retuned) — to be validated in verify.
**Artifact**: `openspec/changes/tui-sidebar-redesign/design.md`

## Flag raised in spec (needs a decision)
The main baseline spec `openspec/specs/caelestia-ui/spec.md` still shows PRE-`tui-visual-redesign` content (opaque `#1a1a2e`, no responsive reqs) — the archived change's main-spec sync never landed. The new delta was written against the effective (post-visual-redesign) behavior so it archives cleanly. Open question: reconcile that stale main-spec sync separately, or ignore for now.

## Confirmed design decisions (Human Review Gate on explore)
- Now-playing: footer card
- Narrow (<90) sidebar: slim rail
- Library: integrated into main content area (slice 3)
- Goldens: add 120x40 tall case (+ existing 60x20/80x24/120x30)
- Defaults (confirm at design): nav items static (no new keys), artwork stacked above lyrics in main

## Preflight decisions
- **Execution mode**: auto (Human Review Gate still mandatory after propose/spec/design/tasks)
- **Artifact store**: OpenSpec
- **Chained PR strategy**: force-chained (encadenados desde el inicio)
- **Review budget**: 400 changed lines per slice
- **Playwright (web tests)**: disabled — Go TUI verified via goldens + asserts

## Request shape (well-formed)
Full UI redesign of the Omusic TUI. Same functionality, no new features.
- **Layout direction**: sidebar (queue/nav, full height) + main content area (artwork + lyrics; library alternates in main area)
- **Aesthetic**: more expressive over the Caelestia palette (mauve/teal) — panel headers with accent bars, richer iconography/state colors, artwork with more presence
- **Dual goal**: use the wasted vertical space AND modernize the flat/dated look
- **Root cause context**: prior `tui-visual-redesign` (archived 2026-07-21) was preservational (translucency + responsive widths) and left the layout/distribution unchanged; panels cap at lyricWindow=12 / maxQueueRows=20 and are top-anchored via `PlaceVertical(..., lipgloss.Top, ...)`, leaving a large empty vertical band on tall terminals.

## Key paths
- `internal/ui/view.go` — layout math, breakpoints, panel rendering
- `internal/ui/styles.go` — Caelestia styles (no opaque Background)
- `internal/ui/testdata/*.golden`, `internal/ui/view_test.go` — fixtures + asserts
- Prior change: `openspec/changes/archive/2026-07-21-tui-visual-redesign/`

## Completed phases
- explore — completed 2026-07-21T08:20:20Z (exploration.md + state.yaml written)

## Explore outcome
- Confirmed empty-vertical-band defect in `testdata/view_120x30.golden` (rows 15–26 blank); root cause = top-anchored `PlaceVertical(..., lipgloss.Top, ...)` + caps 12/20.
- Recommended layout: **Option B** — full-height sidebar + main, now-playing promoted to a footer card, caps raised so `bodyH` becomes real content. Pickers/`modeResults` stay full-screen (driven by `bubbles/list` in `Update`); only library renders into main.
- Proposed 3 chained slices (<400 lines each): (1) structure split, (2) expressive styling, (3) library-in-main.

## Open questions (surfaced at Human Review Gate)
- Now-playing footer card vs. top bar
- Narrow (<90) sidebar: slim rail vs. stacked
- Sidebar nav items decorative vs. static
- Library into main vs. kept full-screen this round
- Artwork stacked vs. beside lyrics
- Golden sizes: keep trio vs. add taller (120x40)

## Design outcome (completed 2026-07-21T09:05:00Z)
- `design.md` written following the archived design's structure. Decisions D1–D8 + edge-case arithmetic.
- D1 sidebar/main split: `sidebarW=clamp(round(split*0.30),26,40)`, `mainW=split-sidebarW`; slim rail <90 = `clamp(round(split*0.22),16,22)`. `split = usable - 2*panelBorder`; `sidebarW+mainW+4==usable` by construction.
- D2 caps lifted: `maxQueueRows=clamp(sidebarH-queueChrome,3,sidebarH)` (queueChrome 5→10 w/ nav in s2); `lyricWindow/plainLines=clamp(mainH-lyricChrome,3,mainH)` (lyricChrome 3→16 when artwork stacked).
- D3/D6: `PlaceVertical` band replaced by `JoinHorizontal(Top, sidebar, main)`, both boxes `.Height(bodyH)`.
- D5: `chromeFixed` 11 (slice1 top-bar) → 14 (slice2 footer card; net +3 = -nowplaying-blank +card4+blank).
- D7 new styles (sidebar/card/navActive/navItem/accentBar) all NO Background; `hasNoBackground`+palette asserts extended.
- D8 library-in-main; pickers/results stay full-screen.
- Slices: 1 Structure ~250–340; 2 Expressive ~270–380 (2a/2b fallback if >400); 3 Library ~160–250. All <400.
- Tests: add 120×40 golden + no-blank-band assert; extend width table `{120,40}`; goldens-differ w/ 120×40; footer-clip 60×20; library-persists.

## Next recommended step
Human Review Gate on design → then `tasks` (archon-tasks).
