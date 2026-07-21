# Exploration: TUI Visual Redesign (`tui-visual-redesign`)

## 1. Executive Summary

Omusic is a Bubble Tea + Lipgloss TUI (`internal/ui`). The user wants a more
visual, "striking" redesign that keeps every existing element, fixes two concrete
defects, and adds a third property they did not name but that the code reveals:

1. **Broken translucency** — opaque `Background(#1a1a2e)` fills on the `title` and
   `panel` styles paint over the terminal's glass/blur effect behind the "Omusic"
   title and the queue/lyrics/artwork panels.
2. **No responsiveness** — panel widths, progress-bar width, truncation limits and
   row/line windows are hardcoded and never derived from `m.width`/`m.height`.
3. **Horizontal overflow (corollary of #2)** — verified: the two golden files
   (`view_80x24.golden` and `view_120x30.golden`) are **byte-identical**. The
   middle section renders ~118 columns wide (36+50+28 panels + borders) in BOTH an
   80-col and a 120-col terminal. In an 80-col terminal it overflows and the
   terminal wraps/clips it; in 120 it leaves dead space on the right.

The redesign must be **purely presentational**: `Model`, `Update`, `messages`,
`keys`, and all services stay untouched. All changes are confined to `styles.go`,
`view.go`, and the golden fixtures. This keeps it well inside the 400-line review
budget for the "minimum" and "medium" options.

Stack (from `go.mod`): `bubbletea v1.3.10`, `lipgloss v1.1.0`, `bubbles v1.0.0`.

---

## 2. Complete UI Element Inventory

### 2.1 Modes (`model.go`, `mode` enum lines 68-80)

| Mode | Render path | Notes |
|---|---|---|
| `modeNormal` | `View()` main layout | Default screen |
| `modeSearch` | `View()` main, `m.input.View()` swapped in for status | Shared text input |
| `modeURLInput` / `modeImportURL` / `modeImportName` / `modeLyricsSearch` | same as search (via `isInputMode()`) | Shared text input with different prompt/placeholder |
| `modeLibrary` / `modeCreatePlaylist` | `renderLibrary()` | Full-screen library view |
| `modePicker` / `modeLyricsPicker` | `m.picker.View()` (bubbles `list`) | Full-screen list, own styling |
| `modeResults` | `m.resultsList.View()` + help line | Full-screen search-results modal, own styling |

### 2.2 Main view elements (`View()`, `view.go` lines 15-69)

1. **Title** — `title.Render("🎵 Omusic")` (bordered box).
2. **Now-playing bar** — `renderNowPlaying()`: state glyph `▶`/`⏸`, truncated title
   (32), progress bar (width 30), `pos/dur`, `vol N`.
3. **Prompt / status line** — `m.input.View()` when `isInputMode()`, else
   `dim.Render(m.status)`.
4. **Middle section** — `renderMiddleSection()`: queue panel + optional enrichment
   (lyrics + artwork), joined with `lipgloss.JoinHorizontal(Top, …)`.
5. **Help line** — `renderHelp()` (single long string of shortcuts).
6. **Bar visualizer** — `renderVisualizer(lipgloss.Width(help))`: animated
   equalizer sized to the help-line width, plays via `animFrame`.
7. Whole block passed through `center()` (horizontal centering only).

### 2.3 Panels

- **Queue** (`renderQueue`, lines 128-163): heading "Cola (N)", sliding window of
  `maxQueueRows=10` rows via `queueWindow`, `▲ N más` / `▼ N más` indicators,
  cache mark `⤓` per row (`cacheMark`), current row marked `▶`. `panel.Width(36)`.
- **Lyrics** (`renderLyricsPanel`, lines 355-401): heading "Letra"; synced lyrics
  show a 7-line window around the active line (`renderSyncedLyrics`) with the
  active line highlighted `▶`; plain lyrics truncated to 48 cols × 8 lines;
  "sin letra" fallback. `panel.Width(50)`.
- **Artwork** (`renderArtworkPanel`, lines 405-415): heading "Portada", ASCII art
  or "[sin portada]". `panel.Width(28)`. Art itself rendered at fixed 24×12
  (`artworkWidth`/`artworkHeight`, `update.go` 1055-1056).

### 2.4 Library view (`renderLibrary`, lines 225-274)

Title "📚 Biblioteca", status/input, tabs `[Playlists] Favoritos Historial`
(active tab bracketed + `selected` style), a cursor list (`renderLibList`, `➤`
marker), help line. Also full-height but **not** width/height aware.

### 2.5 Styles (`styles.go`)

`title, panel, heading, selected, current, dim, help, errorMsg, viz`. Palette:
accent `#e0aaff` (mauve), highlight `#00f5d4` (cyan/teal), dim `#a0a0a0`,
opaque bg `#1a1a2e`.

---

## 3. Translucency Analysis — every opaque paint site

The terminal's glass effect shows through **only where the cell has no background
color**. Any `Background(...)` fills those cells with a solid color, defeating the
blur. Borders are drawn with `BorderForeground` (a glyph on the default cell) and
do **not** themselves fill — the leak is the `Background` on the box body/padding.

| Site | File:line | Kind | Verdict |
|---|---|---|---|
| `title` style `Background(#1a1a2e)` | `styles.go:21` | **Fill** | REMOVE — this is the box behind "🎵 Omusic" |
| `panel` style `Background(#1a1a2e)` | `styles.go:26` | **Fill** | REMOVE — this is the queue/lyrics/artwork body fill; the biggest offender (three panels) |
| `title` border (`RoundedBorder` + `BorderForeground #e0aaff`) | `styles.go:22-23` | Border only | KEEP — glyphs, no fill |
| `panel` border (`RoundedBorder` + `BorderForeground #e0aaff`) | `styles.go:27-28` | Border only | KEEP |
| `Padding(0,1)` on title/panel | `styles.go:24,29` | Padding inherits the style's `Background` | Becomes transparent once `Background` is removed; padding cells then show glass |

Secondary considerations (not opaque fills today, but relevant to the redesign):

- **`bubbles/list` default delegate** (`modePicker`, `modeLyricsPicker`,
  `modeResults`) uses `list.NewDefaultDelegate()` and default `list.Styles`. Its
  selected-item style applies a left border/foreground but the default delegate
  does not paint a solid row background, so the modals are mostly transparent
  already. However the redesign should verify (and, if a themed delegate is added,
  avoid re-introducing) row-background fills.
- **`PlaceHorizontal`** in `center()` (`view.go:119`) pads with **spaces that have
  no background** — those pad cells stay transparent, which is correct. If a future
  option wraps the whole app in a styled container with a background, that would
  re-break translucency across the entire screen; avoid that.

**Conclusion:** removing exactly two `Background(...)` calls
(`styles.go:21` and `styles.go:26`) restores translucency for the title and all
three panels. Foreground accent colors and rounded borders are kept, so the visual
identity (mauve/teal on glass) is preserved. This is a ~2-line change and can ship
independently of the responsive work.

---

## 4. Responsiveness Analysis — every hardcoded dimension

`m.width`/`m.height` are captured on `WindowSizeMsg` (`update.go:22-26`) and today
feed **only** `center()` and the two `list` components' sizes. The main layout
ignores them entirely.

| Hardcoded value | File:line | Should derive from |
|---|---|---|
| Queue panel `Width(36)` | `view.go:140, 162` | share of `m.width` |
| Lyrics panel `Width(50)` | `view.go:368` | share of `m.width` |
| Artwork panel `Width(28)` | `view.go:414` | share of `m.width` (or fixed min for art) |
| Progress bar width `30` | `view.go:198` | `m.width` minus the fixed decorations of the now-playing line |
| Now-playing title truncate `32` | `view.go:202` | derived from available width |
| Queue row truncate `28` | `view.go:156` | queue panel inner width |
| Lyrics plain truncate `48`, lines `8` | `view.go:366` | lyrics panel inner width / available height |
| Synced-lyrics truncate `46`, window `7` | `view.go:373,392` | panel inner width / height |
| `trackLines` truncate `60` (library) | `view.go:309` | `m.width` |
| `maxQueueRows = 10` | `view.go:126` | `m.height` minus header/footer chrome |
| Synced window `7` / plain `8` lines | `view.go:374, 366` | `m.height` |
| Artwork render `24×12` | `update.go:1055-1056` | artwork panel inner width/height |

### 4.1 `center()` limitation

`center()` only centers **horizontally** (`PlaceHorizontal`, `view.go:115-120`).
The vertical space below the visualizer is unused. An "ambitious" option would use
`PlaceVertical`/`Place` and allocate the middle section a proportional slice of the
remaining height.

### 4.2 Suggested breakpoints (columns)

Because the box borders/padding cost ~2 cols per panel and the middle section joins
horizontally, a sensible responsive scheme:

- **Narrow (`width < ~80`)**: single column. Stack queue over enrichment (or hide
  artwork; keep lyrics). Panels take full width minus a small margin.
- **Medium (`~80 ≤ width < ~120`)**: two columns — queue on the left, enrichment
  (lyrics; artwork below or beside) on the right, proportional widths.
- **Wide (`width ≥ ~120`)**: three columns — queue | lyrics | artwork, each a
  fraction of `m.width`, with the artwork column capped so ASCII art stays legible.

Heights: derive `maxQueueRows` and the lyrics window from
`m.height - (title + nowplaying + prompt + help + visualizer + blank lines)`.
The fixed chrome today is roughly 12-14 rows; the remainder is the middle band.

### 4.3 Lipgloss compatibility

All of this is idiomatic Lipgloss v1.1.0: `lipgloss.Width`, `JoinHorizontal`,
`JoinVertical`, `Place/PlaceHorizontal/PlaceVertical`, and computing `Width(n)` from
a runtime integer are all first-class. No new dependency is required. `truncate`
already uses `lipgloss.Width` (display width), so wide/emoji glyphs are handled.

---

## 5. Golden Tests — current behavior and redesign impact

- `TestViewGolden` (`view_test.go:15-45`) renders `View()` at 80×24 and 120×30 with
  a fixed model (2 queue tracks, plain lyrics, artwork, pos=45/dur=180) and compares
  against `testdata/view_<size>.golden`.
- `compareGolden` (lines 49-71) supports **`UPDATE_GOLDEN=1`** to regenerate the
  fixtures; on mismatch it writes a `.got` file and prints a line diff.
- **Current smell**: both golden files are byte-identical (Section 1) — the test
  therefore does NOT actually exercise responsiveness; it only pins the current
  (broken) fixed layout. A redesign that makes layout width-aware will (and should)
  produce **different** output per size, which is the whole point.

**Impact of the redesign on tests:**

1. Both goldens **must be regenerated** (`UPDATE_GOLDEN=1 go test ./internal/ui`).
2. Regeneration alone is not verification — the two fixtures must **differ** after
   the redesign (proof of responsiveness). Recommend adding assertions/new cases:
   - A very narrow case (e.g. 60×20) to lock the single-column breakpoint.
   - An assertion that no rendered line exceeds `m.width` display columns
     (`lipgloss.Width(line) <= width`) — this directly guards the overflow bug and
     is more robust than a byte-for-byte golden across sizes.
3. Translucency (absence of `Background`) is not observable in the plaintext golden
   (ANSI is stripped by the string render in tests). A small unit test on
   `defaultStyles()` asserting `title`/`panel` have no background (e.g. rendering a
   probe and checking for the absence of the bg SGR, or asserting the style's
   `GetBackground() == ""`) would lock defect #1.

See the `go-testing` skill (teatest / golden patterns) for the verify phase.

---

## 6. Redesign Options (2-3 directions) with trade-offs

### Option A — Minimum: transparency + fluid widths (keep structure)

- Remove the two `Background` fills (Section 3).
- Derive the three panel widths, progress-bar width, truncations, `maxQueueRows`
  and lyrics windows from `m.width`/`m.height` (Section 4), keeping today's exact
  layout shape (title, now-playing, status, horizontal middle band, help,
  visualizer).
- Add the narrow breakpoint so the middle band stops overflowing < 80 cols.

**Effort:** ~120-200 changed lines (mostly `view.go` arithmetic + `styles.go` +
regenerated goldens). **Risk:** low. **Lipgloss:** trivial. **PR:** fits one PR
under 400 lines. Delivers exactly what the user explicitly asked for.

### Option B — Medium: new visual hierarchy + responsive column layout

- Everything in A, plus a stronger visual language: section separators/rules,
  accent-colored headings with a subtle underline, a clearer "now-playing" card
  (state + title + progress on distinct visual rows), and a proper responsive
  column system (narrow/medium/wide from Section 4.2) built with `JoinHorizontal`/
  `JoinVertical` helpers.
- Keep every element; only rearrange and restyle.

**Effort:** ~250-380 changed lines. **Risk:** medium (more layout math, more golden
churn, need the extra breakpoints locked by tests). **Lipgloss:** idiomatic. **PR:**
near the 400-line budget — may trigger the ask-always gate; a clean split is
"transparency+fluid" (A) as PR 1, "visual hierarchy" as PR 2.

### Option C — Ambitious: dashboard with proportional regions + vertical fill

- A full dashboard: header band, a proportional multi-region body that uses BOTH
  `m.width` and `m.height` (via `lipgloss.Place`), the visualizer promoted to a
  persistent footer, and the middle band filling the vertical space instead of
  leaving dead rows.
- Possibly a themed `bubbles/list` delegate so the modals/pickers match the new
  look (careful: must not reintroduce opaque row backgrounds — Section 3).

**Effort:** ~400-700+ changed lines. **Risk:** higher (layout regressions across
sizes, golden churn, modal restyle touches picker delegate). **Lipgloss:** still
supported but exercises more of the API. **PR:** exceeds the 400-line budget →
would require chained PRs (`chained-pr` skill). Highest visual payoff, lowest
compatibility margin, most test work.

### Recommendation posture (for propose phase, not decided here)

A is the safe floor and directly satisfies the stated request. B is the sweet spot
for "more visual / striking" without blowing the review budget, ideally shipped as
two slices (A then the hierarchy layer). C is only worth it if the user prioritizes
a bold aesthetic over review size.

---

## 7. Open Questions for the User

1. **Translucency vs. layout priority** — if we must slice this into two PRs
   (400-line budget), do you want translucency-only shipped first (fastest visible
   win), or the responsive layout first?
2. **Boldness level** — Option A (same shape, fixed defects), B (new hierarchy +
   responsive columns), or C (full dashboard using vertical space)?
3. **Palette / theming** — keep the current mauve `#e0aaff` + teal `#00f5d4` accents
   on glass, or introduce a new palette? Should we add a config-driven theme, or is
   a single built-in theme fine for now?
4. **Artwork on narrow terminals** — when there isn't room for three columns, hide
   the artwork panel, shrink it, or move it below? (Lyrics likely takes priority.)
5. **Vertical fill** — should the middle band grow to fill terminal height (Option
   C behavior), or is horizontal responsiveness + a compact centered block enough?
6. **Modals/pickers** — restyle the `bubbles/list` pickers/results modal to match,
   or leave them as-is this round (they are largely transparent already)?
