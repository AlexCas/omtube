# Delta for caelestia-ui

Purely presentational redesign confined to `styles.go`, `view.go`, and the golden
fixtures/tests. `Model`, `Update`, `messages`, `keys`, and services are untouched;
no keybinding or behavior change. This delta layers a full-height **sidebar + main**
structure over the translucent/responsive base from `tui-visual-redesign`.
Requirements are grouped by delivery slice so each ships independently as a chained
PR under 400 changed lines. Slice tags: `@slice1` `@slice2` `@slice3`.

## MODIFIED Requirements

### Requirement: Sidebar + Main Layout (@slice1)

The main view MUST be composed as a full-height left **sidebar** joined to a **main
content area** via `lipgloss.JoinHorizontal`, replacing the former top-anchored
single-row band of loose panels. Both children MUST be exactly `bodyH` rows tall so
the join fills the body with NO top-anchored remainder and NO blank vertical band at
any size. The sidebar hosts the static nav (Cola / Biblioteca / Favoritos /
Historial) and the queue window; the main area stacks artwork ABOVE lyrics.
(Previously: a single-row band `PlaceVertical(bodyH, Top, band)` that pinned panels
to the top and left a blank band at tall sizes.)

#### Scenario: No blank vertical band at 120x40

```gherkin
Scenario: No blank vertical band at 120x40
  Given a terminal 120 columns by 40 rows
  When the main view renders
  Then the sidebar and main area each fill the full body height
  And no fully-blank vertical band appears between the body and the help line
```

#### Scenario: Sidebar and main are joined horizontally

```gherkin
Scenario: Sidebar and main are joined horizontally
  Given a wide terminal with queue, lyrics, and artwork
  When the main view renders
  Then the sidebar (nav + queue) and the main area (artwork above lyrics) render side by side
  And their combined width does not exceed the usable width
```

### Requirement: Layout Resilience (@slice1 @slice2)

The layout MUST be responsive across BOTH width and height. Panel widths,
progress-bar width, truncations, `maxQueueRows`, `lyricWindow`, and `plainLines` MUST
derive from `m.width` / `m.height`, not hardcoded ceilings; the former fixed 12/20
caps MUST be raised or derived from `sidebarH`/`mainH` so content fills the body. NO
rendered line may exceed the terminal width at 60, 80, 120, and 120x40; overflowing
content MUST truncate with ellipsis. Available height MUST be consumed by real
content (queue rows, lyric lines) rather than whitespace, without clipping any
mandatory element.
(Previously: `lyricWindow`/`plainLines` clamped to 12 and `maxQueueRows` to 20,
leaving whitespace at tall sizes; overflow only asserted at 60/80/120.)

#### Scenario: No rendered line exceeds terminal width

```gherkin
Scenario Outline: No rendered line exceeds terminal width
  Given a terminal <width> columns by <height> rows
  When the main view renders
  Then every rendered line has display width <= <width>
  And over-long content is truncated with ellipsis

  Examples:
    | width | height |
    | 60    | 20     |
    | 80    | 24     |
    | 120   | 30     |
    | 120   | 40     |
```

#### Scenario: Content windows derive from body height

```gherkin
Scenario: Content windows derive from body height
  Given the terminal height increases from 30 to 40 rows at 120 columns
  When the main view re-renders
  Then more queue rows and lyric lines become visible
  And no mandatory element (title, now-playing, queue, help, visualizer) is clipped
```

### Requirement: Caelestia Palette & Translucency (@slice1 @slice2)

The system MUST apply the Caelestia accents consistently: accent `#e0aaff` (mauve)
for active/nav/border elements, muted `#a0a0a0` for secondary text, highlight
`#00f5d4` (teal) for selection. The interface MUST stay translucent: NO style may
apply an opaque `Background` fill — this invariant MUST hold for every NEW style
(sidebar box, nav-active, section-header accent bar, now-playing footer card) as well
as `title`, `panel`, the `bubbles/list` delegate row, and the list title. Rounded
borders and foreground accents MUST be preserved. No legacy pink/blue/green colors
MUST remain.
(Previously: translucency asserted only for `title`/`panel` and the delegate; the new
sidebar/card/nav/accent-bar styles did not yet exist.)

#### Scenario: All colors match the Caelestia palette

```gherkin
Scenario: All colors match the Caelestia palette
  When the TUI renders
  Then active/nav elements use accent #e0aaff
  And secondary text uses muted #a0a0a0
  And selection uses highlight #00f5d4
```

#### Scenario: No opaque background on any style including new ones

```gherkin
Scenario: No opaque background on any style including new ones
  Given every constructed style is inspected
  When title, panel, sidebar, nav-active, section-header, footer card, and the list delegate are checked
  Then none exposes any Background color
  And their rounded borders and accents remain
```

### Requirement: Element Parity (@slice1 @slice2 @slice3)

The redesign MUST preserve every existing element, reachable via the SAME modes and
keys: title (`🎵 Omusic` main / `📚 Biblioteca` library); now-playing content
(state glyph `▶`/`⏸`, track title, progress bar, `pos/dur` time, `vol N`); shared
search/prompt/input and status line; queue panel (sliding window, `▲/▼ N más`, cache
mark `⤓`, current `▶`); lyrics panel (synced `▶` highlight + plain fallback +
`sin letra` empty); artwork panel (art or `[sin portada]`, hidden only below the
narrow breakpoint); help line (wrapped); bar visualizer (animated while playing, flat
otherwise); library mode (tabs Playlists/Favoritos/Historial, cursor `➤`,
create-playlist prompt, help); results modal (`modeResults`) and lyric/list pickers.
(Previously: same element set, but distributed as a top-anchored band rather than
sidebar + main.)

#### Scenario: Core main-view elements preserved in sidebar and main

```gherkin
Scenario: Core main-view elements preserved in sidebar and main
  Given a playing track with queue, lyrics, and artwork at a wide size
  When the main view renders
  Then the sidebar shows the nav and queue (window, ▲/▼ N más, ⤓, current ▶)
  And the main area shows artwork above the synced lyrics with the active line highlighted
  And title, now-playing content, status/search, help, and visualizer are all present
```

#### Scenario: Mandatory elements still fit at 60x20

```gherkin
Scenario: Mandatory elements still fit at 60x20
  Given a terminal 60 columns by 20 rows
  When the main view renders
  Then title, now-playing content, queue heading with at least 3 rows, help, and visualizer are all present
  And no rendered line exceeds 60 columns
```

## ADDED Requirements

### Requirement: Now-Playing Footer Card (@slice2)

The now-playing information MUST render as a bordered, accented **footer card**
positioned below the sidebar+main body and above the visualizer, replacing the thin
top now-playing bar. The card MUST preserve full content parity: playback state glyph
(`▶`/`⏸`), truncated track title, progress bar, `pos/dur` time, and `vol N` volume.
The card MUST be translucent (NO opaque `Background`), and `chromeFixed` MUST be
re-measured so the card's added rows do not clip any mandatory element or push a line
past the terminal width.

#### Scenario: Footer card shows now-playing content

```gherkin
Scenario: Footer card shows now-playing content
  Given a playing track with a known position, duration, and volume
  When the main view renders
  Then a bordered accented footer card appears below the body and above the visualizer
  And it shows the ▶ state glyph, track title, progress bar, pos/dur time, and vol N
```

#### Scenario: Footer card does not clip elements at 20 rows

```gherkin
Scenario: Footer card does not clip elements at 20 rows
  Given a terminal 60 columns by 20 rows with the footer card present
  When the main view renders
  Then the title, footer card, queue, help, and visualizer all remain visible
  And no rendered line exceeds 60 columns
```

### Requirement: Accent-Bar Section Headers (@slice2)

Each section heading (sidebar nav header, queue heading, artwork heading, lyrics
heading) MUST render with an accent-colored bar/rule that establishes visual
hierarchy. The accent bar MUST use foreground/border glyphs in the accent color
`#e0aaff` only, with NO `Background` fill. The active nav item MUST be visually
distinguished with the accent color.
Note: the sidebar nav header is FIT-GATED — it renders only when `sidebarH >= 13`
and yields to the mandatory queue window below that height (present at 120x30 and
120x40, absent at 80x24 and 60x20).

#### Scenario: Section headers render with an accent bar

```gherkin
Scenario: Section headers render with an accent bar
  Given the sidebar and main area render at a wide size
  When their section headings are inspected
  Then each heading is preceded or underlined by an accent-colored bar using #e0aaff
  And no accent-bar style exposes a Background color
```

#### Scenario: Active nav item is accented

```gherkin
Scenario: Active nav item is accented
  Given the default main view (queue active)
  When the sidebar nav renders
  Then the active nav item is rendered in the accent color
  And the remaining nav items use the muted color
```

### Requirement: Slim Rail at Narrow Width (@slice1)

Below the narrow breakpoint (< 90 columns) the sidebar MUST collapse to a **slim
rail** rather than stacking above or hiding the main content. The main area MUST keep
its maximum available width so lyrics stay readable. Artwork MUST remain hidden below
the narrow breakpoint (`showArtwork = bp != bpNarrow`), and the queue and lyrics MUST
remain visible.

#### Scenario: Sidebar becomes a slim rail below 90 columns

```gherkin
Scenario: Sidebar becomes a slim rail below 90 columns
  Given a terminal narrower than 90 columns
  When the main view renders
  Then the sidebar renders as a slim rail, not stacked above nor fully hidden
  And the main area (lyrics) keeps its maximum width
  And the artwork panel is not present
  And no rendered line exceeds the terminal width
```

### Requirement: Library In Main (@slice3)

Library mode (`modeLibrary` / `modeCreatePlaylist`) MUST render its content INTO the
main area with the sidebar persisting alongside and the "Biblioteca" nav item
accented, instead of a separate full-screen view. All library elements MUST be
preserved: tabs Playlists/Favoritos/Historial (active bracketed), cursor `➤`,
create-playlist prompt, and the library help line. The `modeResults` modal and the
`modePicker`/`modeLyricsPicker` pickers MUST remain full-screen and MUST NOT couple to
`Update`. Delegate rows in library/pickers MUST stay translucent (no opaque
`Background`).

#### Scenario: Library renders in main with a persistent sidebar

```gherkin
Scenario: Library renders in main with a persistent sidebar
  Given library mode is active
  When the view renders at a wide size
  Then the sidebar persists with "Biblioteca" accented
  And the library tabs, cursor ➤, and help render inside the main area
  And no delegate row applies an opaque background
```

#### Scenario: Results modal and pickers stay full-screen

```gherkin
Scenario: Results modal and pickers stay full-screen
  Given the results modal or a lyric picker is active
  When it renders
  Then it renders full-screen (not inside the sidebar+main body)
  And its rounded borders, accents, and translucent delegate rows are preserved
```

### Requirement: Golden Determinism & Chained Slices (@slice1 @slice2 @slice3)

Golden tests MUST cover 60×20, 80×24, 120×30, AND a new 120×40 case that locks the
vertical-fill behavior. After the redesign the 80×24, 120×30, and 120×40 goldens MUST
differ from one another (proving responsiveness), and the 120×40 golden MUST show no
blank vertical band. The suite MUST assert `lipgloss.Width(line) <= width` for every
rendered line at all four sizes, and MUST assert no `Background` on all new styles,
the delegate, and the list title. Each delivery slice MUST be an independently
shippable chained PR under 400 changed lines.

#### Scenario: 120x40 golden locks vertical fill

```gherkin
Scenario: 120x40 golden locks vertical fill
  Given the sidebar + main layout at 120x40
  When the view renders
  Then it matches the committed 120x40 golden fixture
  And no rendered line exceeds 120 columns
  And the fixture shows no fully-blank vertical band
```

#### Scenario: Goldens differ across sizes

```gherkin
Scenario: Goldens differ across sizes
  Given regenerated golden fixtures
  When the 80x24, 120x30, and 120x40 goldens are compared
  Then no two of them are byte-identical
```
