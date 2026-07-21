# Caelestia UI Specification

## Purpose

Rediseñar la interfaz TUI de Omusic con:
1. **Translucent rendering** — No opaque backgrounds; terminal glass visible throughout.
2. **Responsive layout** — Adapt to terminal width (60/80/120 cols) and height (20/24/30/40 rows).
3. **Full-height sidebar + main** — Sidebar (queue/nav, left side) + main (artwork above lyrics, right side), both filling available height. No blank vertical band.
4. **Expressive styling** — Accent bars on section headers, footer card for now-playing, active nav highlighting.
5. **Library integration** — Library renders in main area during library mode; sidebar navigation persists.
6. **Element parity** — All existing UI elements preserved (queue, lyrics, artwork, now-playing, modals, pickers, library, help, visualizer).
7. **Keyboard preservation** — All existing shortcuts remain functional.

## Palette

| Token | Hex | Role |
|-------|-----|------|
| accent (mauve) | `#e0aaff` | Active/nav elements, borders, highlights, accent bars |
| muted text | `#a0a0a0` | Secondary text, inactive nav items |
| highlight (teal) | `#00f5d4` | Selection, active line highlights |

**Translucency principle**: The interface MUST NOT apply any opaque background fill (no `Background` color in any style). All foreground colors, borders, accents, and glyphs are foreground-driven. This preserves terminal glass and theme visibility.

Applied via `lipgloss` styles in `styles.go`.

## Requirements

### Requirement: Translucent Rendering & Palette

The system MUST apply the Caelestia palette consistently via foreground colors and borders only: accent `#e0aaff` (mauve) for active/nav/accent elements, muted `#a0a0a0` for secondary text, highlight `#00f5d4` (teal) for selection. **The interface MUST be translucent: NO style MAY apply an opaque `Background` fill**. This applies to all new styles (sidebar, footer card, nav items, accent bars, section headers) as well as existing styles (title, panel, delegate rows in modals/pickers). Rounded borders and foreground accents MUST be preserved. No legacy pink/blue/green colors MUST remain.

#### Scenario: All colors match Caelestia palette

When the TUI renders, active/nav elements use accent `#e0aaff`, secondary text uses muted `#a0a0a0`, and selection uses highlight `#00f5d4`.

#### Scenario: No opaque background on any style

Every constructed style (title, panel, sidebar, nav, footer card, section header, modal delegate) exposes NO `Background` color. Rounded borders and accents remain.

### Requirement: Sidebar + Main Full-Height Layout

The main view MUST be composed as a full-height left **sidebar** (hosting queue and navigation) joined to a **main content area** (hosting artwork above lyrics) via `lipgloss.JoinHorizontal`, replacing the former top-anchored single-row band. Both sidebar and main MUST be exactly `bodyH` rows tall so the join fills the body with NO blank vertical band at any size. The sidebar width MUST be 30% of usable width (clamped 26–40 cols) for normal sizes, or 22% (clamped 16–22 cols) below 90 columns (slim rail). Main fills the remainder.

#### Scenario: No blank vertical band at 120x40

At a terminal 120 columns by 40 rows, the sidebar and main area each fill the full body height, and no fully-blank vertical band appears between the body and the help line.

#### Scenario: Sidebar and main are joined horizontally

At any wide size with queue, lyrics, and artwork, the sidebar (nav + queue) and main area (artwork above lyrics) render side by side, and their combined width does not exceed the usable width.

#### Scenario: Sidebar becomes slim rail below 90 columns

At a terminal narrower than 90 columns, the sidebar renders as a slim rail (not stacked, not fully hidden), the main area (lyrics) keeps its maximum width, the artwork panel is not present, and no rendered line exceeds the terminal width.

### Requirement: Responsive Layout Across Width and Height

Panel widths, progress-bar width, truncations, `maxQueueRows`, `lyricWindow`, and `plainLines` MUST derive from `m.width` / `m.height`, not hardcoded constants. NO rendered line may exceed the terminal width at 60, 80, 120, and 120x40 columns; overflowing content MUST truncate with ellipsis. Available height MUST be consumed by real content (queue rows, lyric lines) rather than whitespace, without clipping any mandatory element (title, now-playing, queue, help, visualizer).

#### Scenario Outline: No rendered line exceeds terminal width

| width | height |
|-------|--------|
| 60    | 20     |
| 80    | 24     |
| 120   | 30     |
| 120   | 40     |

At each size, every rendered line has display width ≤ width, and over-long content truncates with ellipsis.

#### Scenario: Content windows derive from body height

When terminal height increases from 30 to 40 rows at 120 columns, more queue rows and lyric lines become visible, and no mandatory element is clipped.

### Requirement: Now-Playing Footer Card

The now-playing information MUST render as a **bordered, accented footer card** positioned between the sidebar+main body and the visualizer/status line, replacing the thin top now-playing bar. The card MUST preserve full parity: playback state glyph (`▶`/`⏸`), truncated track title, progress bar, `pos/dur` time, and `vol N` volume. The card MUST be translucent (NO `Background`), and `chromeFixed` MUST account for the card's rows so no mandatory element is clipped or pushed past the terminal width.

#### Scenario: Footer card shows now-playing content

A playing track renders with a bordered accented footer card below the body and above the visualizer, showing the ▶ state glyph, track title, progress bar, pos/dur time, and vol N.

#### Scenario: Footer card does not clip elements at 20 rows

At a terminal 60 columns by 20 rows, the title, footer card, queue, help, and visualizer all remain visible, and no rendered line exceeds 60 columns.

### Requirement: Accent-Bar Section Headers

Each section heading (sidebar nav header, queue heading, artwork heading, lyrics heading) MUST render with an **accent-colored bar/rule** that establishes visual hierarchy. The accent bar MUST use foreground/border glyphs in the accent color `#e0aaff` only, with NO `Background` fill. The active nav item MUST be visually distinguished with the accent color. The sidebar nav header is **fit-gated**: it renders only when `sidebarH >= 13` and yields to the mandatory queue window below that height (present at 120x30 and 120x40, absent at 80x24 and 60x20).

#### Scenario: Section headers render with an accent bar

At a wide size, each section heading is preceded or underlined by an accent-colored bar using `#e0aaff`, and no accent-bar style exposes a `Background` color.

#### Scenario: Active nav item is accented

In the default main view (queue active), the active nav item renders in the accent color, and remaining nav items use the muted color.

### Requirement: Library In Main

Library mode MUST render its content INTO the main area with the sidebar persisting alongside and the "Biblioteca" nav item accented, instead of a separate full-screen view. All library elements MUST be preserved: tabs Playlists/Favoritos/Historial (active bracketed), cursor `➤`, create-playlist prompt, and the library help line. The `modeResults` modal and the `modePicker`/`modeLyricsPicker` pickers MUST remain full-screen and MUST NOT couple to the main area. Delegate rows in library/pickers MUST stay translucent (no opaque `Background`).

#### Scenario: Library renders in main with persistent sidebar

In library mode at a wide size, the sidebar persists with "Biblioteca" accented, and the library tabs, cursor ➤, and help render inside the main area.

#### Scenario: Results modal and pickers stay full-screen

When a results modal or lyric picker is active, it renders full-screen (not inside the sidebar+main body), and its rounded borders, accents, and translucent delegate rows are preserved.

### Requirement: Element Parity

The redesign MUST preserve every existing element, reachable via the SAME modes and keys: title (`🎵 Omusic` main / `📚 Biblioteca` library); now-playing content (state glyph `▶`/`⏸`, track title, progress bar, `pos/dur` time, `vol N`); shared search/prompt/input and status line; queue panel (sliding window, `▲/▼ N más`, cache mark `⤓`, current `▶`); lyrics panel (synced `▶` highlight + plain fallback + `sin letra` empty); artwork panel (art or `[sin portada]`, hidden only below the narrow breakpoint); help line (wrapped); bar visualizer (animated while playing, flat otherwise); library mode (tabs Playlists/Favoritos/Historial, cursor `➤`, create-playlist prompt, help); results modal (`modeResults`) and lyric/list pickers.

#### Scenario: Core main-view elements preserved in sidebar and main

A playing track with queue, lyrics, and artwork at a wide size renders the sidebar showing nav and queue (window, ▲/▼ N más, ⤓, current ▶), and the main area showing artwork above synced lyrics with active line highlighted. Title, now-playing content, status/search, help, and visualizer are all present.

#### Scenario: Mandatory elements fit at 60x20

At a terminal 60 columns by 20 rows, the title, now-playing content, queue heading with at least 3 rows, help, and visualizer are all present, and no rendered line exceeds 60 columns.

### Requirement: Golden Determinism

Golden tests MUST cover 60×20, 80×24, 120×30, AND 120×40 (locking vertical-fill behavior). The 80×24, 120×30, and 120×40 goldens MUST differ from one another (proving responsiveness), and the 120×40 golden MUST show no blank vertical band. The test suite MUST assert `lipgloss.Width(line) <= width` for every rendered line at all sizes, and MUST assert no `Background` on all styles (title, panel, sidebar, nav, footer card, section headers, delegate). Width assertions MUST cover 60, 80, 120 columns at their primary heights.

#### Scenario: 120x40 golden locks vertical fill

At 120x40, the sidebar + main layout matches the committed 120x40 golden fixture. No rendered line exceeds 120 columns. The fixture shows no fully-blank vertical band.

#### Scenario: Goldens differ across sizes

The 80×24, 120×30, and 120×40 golden fixtures are NOT byte-identical, proving that responsiveness is deterministic across sizes.

### Requirement: Keyboard Shortcut Preservation

All existing keyboard shortcuts (space, `n`, `p`, `+`/`-`, `/`, `q`, library/favorites actions) MUST remain functional after the redesign. No shortcut mapping MUST change.

#### Scenario: All existing keyboard shortcuts still work

All existing keyboard shortcuts remain functional after the redesign.
