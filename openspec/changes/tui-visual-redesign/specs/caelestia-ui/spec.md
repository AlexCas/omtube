# Delta for caelestia-ui

Purely presentational redesign confined to `styles.go`, `view.go`, and the golden
fixtures/tests. `Model`, `Update`, `messages`, `keys`, and services are untouched;
no keybinding or behavior changes. Requirements are grouped by delivery slice so
Slice 1 ships independently. Slice tags: `@slice1` `@slice2` `@slice3`.

## MODIFIED Requirements

### Requirement: Caelestia Palette (@slice1)

The system MUST apply the Caelestia accents consistently: accent `#e0aaff` (mauve)
for active/border elements, muted `#a0a0a0` for secondary text, highlight `#00f5d4`
(teal) for selection. The interface MUST be translucent by default: NO style may
apply an opaque `Background` fill to `title` or `panel` (nor to any `bubbles/list`
delegate row used by modals/pickers). Rounded borders and foreground accents MUST be
preserved. No legacy pink/blue/green colors MUST remain.
(Previously: required an opaque deep-blue `#1a1a2e` background on panels/title.)

#### Scenario: All colors match Caelestia palette after redesign

```gherkin
Scenario: All colors match Caelestia palette after redesign
  When the TUI renders
  Then active elements use accent #e0aaff
  And secondary text uses muted #a0a0a0
  And selection uses highlight #00f5d4
```

#### Scenario: No opaque background paints over the terminal glass

```gherkin
Scenario: No opaque background paints over the terminal glass
  Given the default styles are constructed
  When the title and panel styles are inspected
  Then neither style exposes any Background color
  And their rounded borders remain
```

### Requirement: Layout Resilience (@slice1 @slice2)

The layout MUST be responsive across BOTH width and height. Panel widths,
progress-bar width, truncations, `maxQueueRows`, and lyric windows MUST derive from
`m.width` / `m.height`, not hardcoded constants. NO rendered line may exceed the
terminal width at representative sizes (60, 80, 120 columns); overflowing content
MUST truncate with ellipsis. A narrow breakpoint MUST stop the 80-column overflow.
Vertical space (`m.height`) MUST be used to size regions without clipping any
mandatory element. At larger sizes, regions MUST expand proportionally.
(Previously: only asserted no overflow at 80×24 with hardcoded panel widths.)

#### Scenario: No rendered line exceeds terminal width (@slice1)

```gherkin
Scenario Outline: No rendered line exceeds terminal width
  Given a terminal <width> columns wide
  When the main view renders
  Then every rendered line has display width <= <width>
  And over-long content is truncated with ellipsis

  Examples:
    | width |
    | 60    |
    | 80    |
    | 120   |
```

#### Scenario: Widths derive from runtime dimensions (@slice1)

```gherkin
Scenario: Widths derive from runtime dimensions
  Given the terminal width changes from 80 to 120 columns
  When the main view re-renders
  Then queue, lyrics, and artwork widths and truncations change accordingly
```

#### Scenario: Vertical space is used without clipping (@slice2)

```gherkin
Scenario: Vertical space is used without clipping
  Given a terminal 120 columns by 30 rows
  When the main view renders
  Then content is placed using the available height
  And no mandatory element (title, now-playing, queue, help, visualizer) is clipped
```

## ADDED Requirements

### Requirement: Responsive Breakpoints (@slice2)

The system MUST select a layout breakpoint from `m.width`: narrow (`< ~80`),
medium (`~80–120`), and wide (`≥ ~120`). Each breakpoint MUST render a distinct,
deterministic column distribution. Under the narrow breakpoint the artwork panel
MUST be hidden (not shrunk or moved); the queue and lyrics MUST remain visible.

#### Scenario: Narrow breakpoint hides artwork

```gherkin
Scenario: Narrow breakpoint hides artwork
  Given a terminal narrower than the narrow breakpoint
  When the main view renders
  Then the artwork panel is not present
  And the queue and lyrics panels are still rendered
```

#### Scenario: Breakpoints render distinct deterministic layouts

```gherkin
Scenario: Breakpoints render distinct deterministic layouts
  Given identical model state
  When the view renders at 60, 80, and 120 columns
  Then the three outputs differ from each other
  And each output is deterministic for its size
```

### Requirement: Element Parity (@slice1 @slice2 @slice3)

The redesign MUST preserve every existing element: title (`🎵 Omusic` /
`📚 Biblioteca`); now-playing bar (state glyph, title, progress, time, volume);
shared search/prompt/input and status line; queue panel (sliding window, `▲/▼ N más`,
cache mark `⤓`, current `▶`); lyrics panel (synced highlight + plain fallback);
artwork panel (hidden only below the narrow breakpoint); help line; bar visualizer;
library mode (tabs Playlists/Favoritos/Historial, cursor, create playlist); results
modal (`modeResults`); and lyric/list pickers.

#### Scenario: Core main-view elements preserved (@slice1)

```gherkin
Scenario: Core main-view elements preserved
  Given a playing track with queue, lyrics, and artwork
  When the main view renders at a wide size
  Then title, now-playing bar, status/search, queue, lyrics, artwork, help, and visualizer are all present
```

#### Scenario: Queue and lyrics behaviors preserved (@slice2)

```gherkin
Scenario: Queue and lyrics behaviors preserved
  Given a queue longer than the visible window and synced lyrics
  When the panels render
  Then the queue shows the sliding window with ▲/▼ N más, cache mark ⤓, and current ▶
  And the lyrics panel highlights the active line with the highlight color
```

#### Scenario: Modals, library, and pickers preserved and translucent (@slice3)

```gherkin
Scenario: Modals, library, and pickers preserved and translucent
  Given the results modal, library view, or a lyric picker is active
  When it renders
  Then its tabs, cursor, and controls are present
  And no delegate row applies an opaque background
  And rounded borders and accents remain
```

### Requirement: Golden Determinism (@slice1 @slice2)

Golden tests MUST cover 60×20, 80×24, and 120×30. After the redesign the 80×24 and
120×30 goldens MUST differ (proving responsiveness), and a 60×20 golden MUST lock the
narrow single-column breakpoint. The suite MUST assert `lipgloss.Width(line) <= width`
for every rendered line and MUST assert `title`/`panel` styles expose no `Background`.

#### Scenario: 80×24 and 120×30 goldens differ (@slice1)

```gherkin
Scenario: 80×24 and 120×30 goldens differ
  Given regenerated golden fixtures
  When the 80×24 and 120×30 goldens are compared
  Then they are not byte-identical
```

#### Scenario: Narrow 60×20 golden is locked (@slice2)

```gherkin
Scenario: Narrow 60×20 golden is locked
  Given the narrow breakpoint layout
  When the view renders at 60×20
  Then it matches the committed 60×20 golden fixture
  And no rendered line exceeds 60 columns
```
