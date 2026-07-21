# Exploration: TUI Sidebar Redesign (`tui-sidebar-redesign`)

## 0. Framing correction (why this change exists)

The prior change `tui-visual-redesign` (archived `2026-07-21-tui-visual-redesign`)
delivered a **cosmetic** pass: it removed the two opaque `Background(#1a1a2e)` fills
(restoring translucency), made panel widths/truncations/windows fluid from
`m.width`/`m.height`, and added the narrow/medium/wide breakpoints. What it did NOT
do is change the on-screen **distribution**: the layout is still
title → now-playing → status → one horizontal band of loose panels → help →
visualizer, all anchored to the TOP of the terminal.

The team read "preserve elements" as "preserve the layout/distribution". The user
actually meant **"do not invent new functionality"** — the layout and aesthetic
SHOULD change substantially. This change is the layout/aesthetic redesign the user
originally wanted, while keeping the same behavior surface.

**Hard constraint (unchanged from last time):** purely presentational. NO changes to
`Model`, `Update`, `messages`, `keys`, or services; no new features, no new
keybindings. All edits confined to `internal/ui/styles.go`, `internal/ui/view.go`,
`internal/ui/view_test.go`, and `internal/ui/testdata/*.golden`.

Stack (from `go.mod`): `bubbletea v1.3.10`, `lipgloss v1.1.0`, `bubbles v1.0.0`.
No new dependency required — `JoinHorizontal`/`JoinVertical`/`Place`/`PlaceVertical`
are all first-class in lipgloss v1.1.0.

---

## 1. Executive Summary

Two defects the user named, both traceable to the current render pipeline:

1. **Wasted vertical space** — verified in `testdata/view_120x30.golden`: rows 15–26
   (twelve rows) are blank between the panel band and the help line. Root cause is
   the combination of top-anchoring in `renderMiddleSection`
   (`lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)`, `view.go:527`) plus the
   window caps `maxQueueRows := clamp(bodyH-5, 3, 20)` and
   `lyricWindow := clamp(bodyH-3, 3, 12)` / `plainLines := clamp(bodyH-3, 3, 12)`
   (`view.go:136,139,153`). The panels stop growing at their caps, `PlaceVertical`
   pins them to the top of `bodyH`, and every remaining row is whitespace.
2. **Flat / dated "loose list of panels" look** — three equally-weighted bordered
   boxes sit in a single row with no visual hierarchy, no persistent navigation, no
   sense of a primary content region. Headings are plain bold text with no accent
   bar; artwork is a small third column with no visual presence.

The user's chosen direction: a **sidebar + main content** layout. A fixed
full-height left sidebar for queue/navigation (Cola, Biblioteca, Favoritos,
Historial); a main content area showing artwork + synced lyrics, with the
library/pickers rendering in that main area. The now-playing bar, progress, volume,
visualizer, and help remain. Aesthetic goes **more expressive** over the existing
Caelestia palette (mauve `#e0aaff` / teal `#00f5d4`): panel headers with accent
bars, richer iconography and state colors, artwork with more presence. Translucency
invariant holds (no opaque `Background` fills).

Both defects are addressed by one structural move: the **sidebar is full-height**
(it fills `bodyH` by construction), and the **main area sizes its content windows to
`bodyH`** instead of the 12/20 caps. Vertical space is consumed by real content, not
whitespace.

---

## 2. Current UI rendering pipeline (survey)

### 2.1 Entry point — `View()` (`view.go:188-246`)

Dispatch order:

1. `m.quitting` → farewell string.
2. `modePicker` / `modeLyricsPicker` → `themedList(m.picker).View()` (full-screen
   bubbles/list).
3. `modeResults` → `themedList(m.resultsList).View()` + a help line (full-screen
   modal).
4. `l := computeLayout(m.width, m.height)` computed once.
5. `modeLibrary` / `modeCreatePlaylist` → `renderLibrary(l)` (full-screen).
6. Default (`modeNormal` + input modes): build the main block —
   - `title.Render("🎵 Omusic")` (bordered box) + blank line
   - `renderNowPlaying(l)` + blank line
   - input view (if `isInputMode()`) else `dim.Render(m.status)` + blank line
   - `renderMiddleSection(l)` + blank line
   - `renderHelp()` + newline
   - `renderVisualizer(lipgloss.Width(help))` + newline
   - whole block through `center()` (**horizontal** centering only).

### 2.2 Layout math — `computeLayout` (`view.go:79-158`)

Pure function `(width, height) → layout`. Key derivations:

- `classify(width)` → `bpNarrow` (<90) / `bpMedium` (90–119) / `bpWide` (≥120).
- `usable = max(width-2, minUsable=40)`.
- Vertical chrome: `chromeFixed = 11` rows + `helpRows(width)` (wrapped help height);
  `bodyH = max(height - (chromeFixed + helpRows), minBody=4)`.
- Column widths: narrow splits `usable` between queue (~42%) and lyrics (~58%);
  medium/wide use three columns (queue 34/30%, artwork 26%, lyrics remainder) with
  min clamps (`qMin=24, lMin=28, aMin=24, aMax=28`) and remainder folded into lyrics.
- `progressW = clamp(width - nowDecor(24) - nowTitleTrunc, 8, 40)`.
- **The caps that leave whitespace**: `maxQueueRows = clamp(bodyH-5, 3, 20)`,
  `lyricWindow = clamp(bodyH-3, 3, 12)` (odd-normalized), `plainLines = clamp(bodyH-3, 3, 12)`.
- `showArtwork = bp != bpNarrow`.

### 2.3 Middle band — `renderMiddleSection` (`view.go:522-528`)

```go
band := m.renderQueueAt(l)
if enrich := m.renderEnrichment(l); enrich != "" {
    band = lipgloss.JoinHorizontal(lipgloss.Top, band, enrich)
}
return lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)
```

`renderEnrichment` (`view.go:534-548`) joins lyrics + artwork panels (each optional,
gated on `m.lyrics != nil` / `m.artwork != nil && l.showArtwork`). This is the
"loose row of panels" and the top-anchor that produces the empty band.

### 2.4 Panel renderers

- `renderQueueAt(l)` (`view.go:311-346`): heading `Cola (N)`, `queueWindow` sliding
  window of `maxQueueRows`, `▲/▼ N más` markers, per-row cache mark `⤓`
  (`cacheMark`), current row `▶`. Wrapped in `panel.Width(l.queueW)`.
- `renderLyricsPanelAt(l)` (`view.go:558-572`): heading `Letra`; synced window
  (`renderSyncedLyrics`, `▶` on active line) / plain truncated / `sin letra`. Wrapped
  in `panel.Width(l.lyricsW)`.
- `renderArtworkPanelAt(l)` (`view.go:615-625`): heading `Portada`; `m.curArtwork` or
  `[sin portada]`. Wrapped in `panel.Width(l.artW)`. Art itself is rendered upstream
  at fixed 24×12 in `update.go` (NOT touched — outside the presentational surface).
- `renderNowPlaying(l)` (`view.go:372-389`): `▶`/`⏸` glyph, truncated title
  (teal `current`), `progressBar`, `pos/dur`, `vol N`. Single line.
- `renderVisualizer(width)` (`view.go:255-270`): animated equalizer sized to help
  width; flat when not playing.
- `renderLibrary(l)` (`view.go:416-465`): full-screen. Title `📚 Biblioteca`,
  status/input, tabs `[Playlists] Favoritos Historial` (active bracketed +
  `selected`), a cursor list (`renderLibList`, `➤`), help line, `center()`.

### 2.5 Styles — `styles.go`

`title, panel, heading, selected, current, dim, help, errorMsg, viz`.
Palette after the prior change: accent `#e0aaff` (mauve), highlight `#00f5d4`
(teal), muted `#a0a0a0`. `title`/`panel` are rounded-bordered with NO `Background`
(translucency invariant). `caelestiaListDelegate()` themes the modal/picker list
(foreground/border only, no row background) and `themedList()` replaces the default
`list.Styles.Title` background with a translucent mauve title.

---

## 3. Parity checklist — every element that MUST survive (behavior parity)

Reused and extended from the prior proposal's checklist. Each item is presentational
and MUST remain functionally reachable (same modes, same keys, same content):

- [ ] Title "🎵 Omusic" (main) / "📚 Biblioteca" (library)
- [ ] Now-playing bar: `▶`/`⏸` state glyph, track title, progress bar, `pos/dur`
      time, `vol N` volume
- [ ] Shared search/prompt input (search, URL, import URL, import name, lyrics search)
- [ ] Status line (`m.status`)
- [ ] Queue panel: heading `Cola (N)`, sliding window (`queueWindow`), `▲/▼ N más`
      markers, per-row cache mark `⤓`, current-row `▶`
- [ ] Lyrics panel: synced window with `▶` active-line highlight; plain fallback;
      `sin letra` empty state
- [ ] Artwork panel: rendered art or `[sin portada]`; hidden below the width
      breakpoint (never shrunk/moved into an illegible size)
- [ ] Help line (wrapped to width)
- [ ] Bar visualizer (animated while playing, flat when paused/stopped)
- [ ] Library mode: tabs Playlists/Favoritos/Historial, cursor `➤`, create-playlist
      prompt, its own help line
- [ ] Results modal (`modeResults`) + pickers (`modePicker`, `modeLyricsPicker`)
- [ ] Translucency: NO opaque `Background` on any style, delegate row, or list title
- [ ] Caelestia palette assertions (`#e0aaff` / `#00f5d4` / `#a0a0a0`)
- [ ] No rendered line exceeds terminal width at 60/80/120 columns

Behavior parity guardrails (must not regress): `TestClassifyBoundaries`,
`TestComputeLayoutWidths`, `TestComputeLayoutHeight`, `TestStylesNoBackground`,
`TestDelegateNoBackground`, `TestLibraryViewIsTranslucent`, `TestCaelestiaAccentColors`,
`TestNoLineExceedsWidth`, `TestGoldensDiffer`, and the toggle-off parity tests in
`update_test.go`. Some of these WILL need their expected constants retuned (e.g.
column split percentages, `bodyH` chrome) — that is intended churn, not a behavior
change. `hasNoBackground` and the palette asserts MUST keep passing verbatim.

---

## 4. What fills the vertical space (the fix)

The empty band disappears when two things change together:

1. **Full-height sidebar.** The left sidebar is rendered to occupy `bodyH` rows by
   construction (a bordered box whose inner content is `PlaceVertical`-filled or
   whose height is set so the border bottom lands at `bodyH`). Because it is
   full-height, there is no whitespace below it — the sidebar border reaches the
   help line.
2. **Main area sizes to `bodyH`, not to the 12/20 caps.** `maxQueueRows`,
   `lyricWindow`, and `plainLines` are recomputed against the sidebar/main heights
   with the artificial ceilings raised (or removed) so lyrics fill the main region
   and the queue fills the sidebar region. The remainder that used to be whitespace
   becomes visible lyric lines / queue rows / artwork breathing room.

Concretely, `renderMiddleSection`'s `PlaceVertical(bodyH, Top, band)` is replaced by
a `JoinHorizontal(Top, sidebar, main)` where BOTH children are already `bodyH` tall,
so the join is exactly `bodyH` rows with no top-anchored remainder.

---

## 5. How sidebar + main restructures `computeLayout` / `View` / renderers

### 5.1 `computeLayout`

- Add `sidebarW` and `mainW` (outer widths) replacing the queue/lyrics/artwork triad
  as the top-level split. Sidebar is a fixed-ish fraction (e.g. ~28–34% of `usable`,
  clamped `[qMin, ~40]`); main takes the remainder.
- Within `mainW`, keep an internal lyrics/artwork split (artwork gated by
  `showArtwork`) — the existing three-way clamp logic largely survives, re-parented
  under `mainW` instead of `usable`.
- Add `sidebarH` / `mainH` (= `bodyH`) so renderers can build full-height boxes.
- Raise/remove the `12`/`20` caps: `maxQueueRows` derives from `sidebarH - chrome`
  (queue heading + nav section + markers); `lyricWindow`/`plainLines` derive from
  `mainH - chrome`. Keep the odd-normalization on `lyricWindow` and the `minBody`
  floor so nothing collapses at 20 rows.
- `nav` section: the sidebar hosts a static nav list (Cola / Biblioteca / Favoritos /
  Historial). These labels already correspond to reachable modes/sections — rendering
  them as a nav header is presentational; the KEYS that switch modes are unchanged.

### 5.2 `View`

- The default branch builds `sidebar := m.renderSidebar(l)` and
  `main := m.renderMain(l)`, joins them `JoinHorizontal(Top, sidebar, main)`, and the
  now-playing bar + progress + volume can either stay above the split (top bar) or
  move into a footer card — see options in §6.
- `renderLibrary` renders its list INTO the main area (sidebar persists with
  "Biblioteca" highlighted) rather than a separate full-screen view — this is the
  "library/pickers render in the main area" requirement. Pickers/results modal can
  stay full-screen (simplest, lowest risk) OR render in main; recommend keeping the
  full-screen modal for `modeResults`/pickers this round to bound scope (they are
  transient overlays, not persistent navigation).

### 5.3 Renderers

- New `renderSidebar(l)`: nav header (Cola/Biblioteca/Favoritos/Historial with the
  active one accented) + the queue window, wrapped in a full-height bordered box.
- New `renderMain(l)`: artwork (with more presence) + synced/plain lyrics, wrapped in
  a full-height region. `renderLibrary` content renders here when in library mode.
- `renderQueueAt` / `renderLyricsPanelAt` / `renderArtworkPanelAt` are retained but
  re-scoped to sidebar/main widths and heights; their historic no-arg wrappers
  (`renderQueue`, `renderLyricsPanel`, `renderArtworkPanel`, called by tests) stay.
- Accent-bar headers: a small helper (e.g. `sectionHeader(label)`) renders an
  accent-colored bar/rule under each heading — foreground/border glyphs only, no
  `Background`.

---

## 6. Layout options (with tradeoffs) + recommendation

### Option A — Sidebar + main, now-playing stays as a top bar

- Structure: top now-playing bar (as today) → below it, `JoinHorizontal(sidebar,
  main)` filling `bodyH` → help → visualizer.
- Sidebar: full-height, nav header + queue. Main: full-height, artwork + lyrics.
- Effort: smallest of the three; the top bar and chrome math barely move. Golden
  churn moderate. Risk low.
- Tradeoff: the now-playing bar is still a thin top line — less "card-like" presence,
  but the vertical band is gone and the sidebar/main hierarchy lands.

### Option B — Sidebar + main, now-playing promoted to a footer card (RECOMMENDED)

- Structure: title header → `JoinHorizontal(sidebar, main)` filling most of `bodyH`
  → a now-playing **card** (state + title on one row, progress + times + volume on a
  second row, inside a translucent bordered/accented card) → help → visualizer.
- Sidebar: full-height, accent-bar nav header (Cola/Biblioteca/Favoritos/Historial,
  active accented) + queue window sized to sidebar height. Main: full-height,
  artwork with more presence stacked above (or beside, wide only) the synced/plain
  lyrics; accent-bar headers on each region.
- Effort: medium. Now-playing goes from 1 to ~2 rows and moves into a card
  (re-measures `chromeFixed`). Golden churn is the largest here.
- Risk: medium (chrome re-measure, breakpoint retune, golden regeneration) — all
  contained by the width/height asserts and `TestNoLineExceedsWidth`.
- **Why recommended:** it satisfies BOTH stated priorities with equal weight — the
  full-height sidebar+main eliminates the wasted band, and the footer now-playing
  card + accent-bar headers deliver the "more expressive, less flat" modernization.
  It is the most consistent with the user's explicit "sidebar + main + more
  expressive" choice while staying purely presentational and within budget when
  sliced.

### Option C — Sidebar + main + main-area library/pickers + full modal restyle

- Everything in B, plus rendering `modeResults` and the pickers INTO the main area
  (sidebar persists) instead of full-screen, with a bespoke non-`list` renderer.
- Effort: large. Reworking the modal/picker away from `bubbles/list` risks touching
  `Update` (list navigation is driven there) — brushes against the "no `Update`"
  constraint and the parity guardrails.
- Risk: high (parity, budget, possible `Update` coupling). Not recommended this
  round.

**Recommendation: Option B.** Keep `modeResults`/pickers full-screen (as today) to
avoid `Update` coupling; render only the **library** into the main area (it is a
persistent navigation destination, already width/height-aware via `renderLibrary`).

---

## 7. Responsive behavior (what the sidebar does at narrow widths)

- **Wide (≥120):** sidebar (nav + queue) | main (artwork beside lyrics, or artwork
  stacked above lyrics). Full three-region richness.
- **Medium (90–119):** sidebar | main (artwork stacked ABOVE lyrics inside main, not
  a third column) — keeps two top-level regions, artwork retains presence without a
  third border column.
- **Narrow (<90):** the sidebar must NOT crowd out the main content. Two viable
  behaviors, to be decided at propose/design:
  1. **Collapse to a slim nav rail** — sidebar shrinks to a minimal fixed width
     (icons/short labels + a compact queue), main takes the rest. Preserves the
     sidebar identity at all widths.
  2. **Drop the sidebar border into a single stacked column** — sidebar content
     (nav + queue) stacks ABOVE main content (lyrics; artwork already hidden < 90 per
     existing `showArtwork`). Matches the existing narrow philosophy (hide/stack, not
     shrink-to-illegible).
  Recommendation leans to (2) for consistency with the current narrow rule and the
  existing `Test60x20NarrowNoArtwork` invariant (artwork hidden < 90), but (1) better
  preserves the "sidebar" identity — an open question for the user.
- Artwork stays hidden below 90 (existing `showArtwork = bp != bpNarrow` invariant
  and `Test60x20NarrowNoArtwork`).

---

## 8. Translucency invariant & no-`Model`/`Update` constraint (call-outs)

- **Translucency:** every new style (accent bars, now-playing card, sidebar box, nav
  header) MUST be foreground/border only. NO `Background(...)` anywhere. The existing
  `hasNoBackground` helper and `TestStylesNoBackground` / `TestDelegateNoBackground` /
  `TestLibraryViewIsTranslucent` are the guardrails; extend them to any NEW style
  struct fields (e.g. a `card` or `navActive` style) so the invariant is locked for
  the additions too.
- **No `Model`/`Update`/messages/services/keys changes:** the sidebar nav is a
  presentational reflection of existing modes/sections. Switching between Cola /
  Biblioteca / Favoritos / Historial uses the SAME keys that exist today (`L` for
  library, `n/p` for library sections). No new key bindings, no new `Model` fields,
  no `Update` cases. If a design temptation arises to add sidebar-focus navigation,
  that is OUT OF SCOPE for this change.

---

## 9. Proposed chained-slice breakdown (each < 400 changed lines)

Ordered so each slice is independently shippable and revertible (none touch
`Model`/`Update`/services). Line estimates include golden regeneration.

| Slice | Scope | Files | Est. lines | Deliverable at end |
|-------|-------|-------|-----------|--------------------|
| **1 Structure** | Introduce sidebar + main split in `computeLayout` (`sidebarW`/`mainW`/`sidebarH`/`mainH`), replace `renderMiddleSection`'s top-anchored band with `JoinHorizontal(Top, sidebar, main)` where both are full-height; raise/remove the 12/20 caps so content fills `bodyH`. Keep now-playing as a top bar (Option A intermediate). Retune width/height asserts; regenerate goldens. | `view.go`, `view_test.go`, 3 goldens | ~220–320 | Empty vertical band GONE; sidebar/main hierarchy present; all invariants pass |
| **2 Expressive styling** | Accent-bar section headers (`sectionHeader` helper), sidebar nav header (Cola/Biblioteca/Favoritos/Historial, active accented), now-playing promoted to a footer **card** (2-row, bordered, accented) → re-measure `chromeFixed`; richer state colors/iconography; artwork with more presence (stacked in main). New styles added to `styles` struct (all no-`Background`); extend `hasNoBackground` asserts to them. Regenerate goldens. | `styles.go`, `view.go`, `view_test.go`, 3 goldens | ~250–360 | "More expressive" aesthetic lands; card footer; nav header; translucency asserted on new styles |
| **3 Library in main** | Render `renderLibrary` content INTO the main region (sidebar persists with "Biblioteca" accented) instead of full-screen; keep `modeResults`/pickers full-screen. Add a library-in-main golden + assert sidebar persists. Regenerate goldens. | `view.go`, `view_test.go`, goldens | ~150–240 | Library renders in main area with persistent sidebar |

Re-slicing forecast: Slice 2 is the largest risk (card footer + accent headers +
golden churn). If it forecasts past 400 at tasks phase, split into 2a (accent-bar
headers + sidebar nav) and 2b (now-playing footer card + chrome re-measure). No
current forecast crosses 400.

---

## 10. Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Golden churn masks a real regression | High | Regenerate deliberately per slice; lean on `TestNoLineExceedsWidth`, `hasNoBackground`, and palette asserts rather than only byte goldens; review each `.got` diff before committing |
| `bubbles/list` delegate / library-in-main re-introduces opaque row background | Med | Keep foreground/border-only styling; extend `TestDelegateNoBackground` / `TestLibraryViewIsTranslucent`; do NOT restyle pickers away from `bubbles/list` (avoids `Update` coupling) |
| Breakpoint edge cases (89/90, 119/120) break the sidebar/main split | Med | Retune and re-assert `TestClassifyBoundaries` + `TestComputeLayoutWidths` on boundary widths; assert `sidebarW + mainW <= usable` at each boundary |
| Chrome re-measure (footer card) drifts `bodyH`, clipping mandatory elements at 20 rows | Med | Keep `minBody` floor + row mins; add/keep a 60×20 case asserting title, now-playing, queue heading + ≥3 rows, help, and visualizer all present |
| Slice 2 exceeds 400-line budget | Med | Pre-planned 2a/2b split; enforce at tasks phase |
| Sidebar at narrow widths crowds out lyrics | Med | Decide narrow behavior (slim rail vs. stacked) at propose; assert no line exceeds width at 60/80 and lyrics remain present |
| "More expressive" is subjective — apply may over/under-shoot | Low | Lock concrete, testable aesthetic decisions in design (accent-bar glyph, card border style, nav marker); goldens pin the exact output |

---

## 11. Open questions (for the Human Review Gate)

1. **Now-playing placement** — footer card (Option B, recommended) vs. keep it as a
   top bar (Option A)? The card is more expressive but is the biggest chrome/golden
   change.
2. **Narrow (<90) sidebar behavior** — slim nav rail (preserves sidebar identity) or
   stacked single column (consistent with the existing hide/stack narrow rule)?
3. **Sidebar nav semantics** — should the nav items (Cola/Biblioteca/Favoritos/
   Historial) be purely decorative labels reflecting current mode, or is showing them
   as a static list acceptable given they map to existing keys? (No new keys either
   way — confirming they stay presentational.)
4. **Library scope** — render library INTO main (Slice 3, recommended) or keep it
   full-screen this round to shrink scope? Pickers/results modal stay full-screen
   regardless (to avoid `Update` coupling) — confirm.
5. **Artwork presentation in main** — stacked ABOVE lyrics at all non-narrow widths,
   or beside lyrics at wide only? Affects `mainW` internal split.
6. **Golden sizes** — keep the existing 60×20 / 80×24 / 120×30 trio, or add a taller
   case (e.g. 120×40) to lock the vertical-fill behavior explicitly?
