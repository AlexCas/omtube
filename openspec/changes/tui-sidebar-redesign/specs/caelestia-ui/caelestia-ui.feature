Feature: Caelestia UI Sidebar Redesign
  Omusic's TUI is restructured into a full-height sidebar (nav + queue) joined to a
  main content area (artwork above lyrics), with the now-playing bar promoted to a
  footer card and a more expressive Caelestia (mauve/teal) treatment. Purely
  presentational: no Model/Update/messages/keys/services changes, no new keybindings,
  translucency preserved. Delivered in three chained slices, each a PR under 400
  changed lines; scenarios are tagged @slice1, @slice2, @slice3. Go TUI verified via
  golden fixtures and asserts (no Playwright).

  Background:
    Given Omusic is running with a playing track, a queue, lyrics, and artwork

  @slice1 @happy
  Scenario: Sidebar and main are joined horizontally
    Given a wide terminal with queue, lyrics, and artwork
    When the main view renders
    Then the sidebar (nav + queue) and the main area (artwork above lyrics) render side by side
    And their combined width does not exceed the usable width

  @slice1 @edge
  Scenario: No blank vertical band at 120x40
    Given a terminal 120 columns by 40 rows
    When the main view renders
    Then the sidebar and main area each fill the full body height
    And no fully-blank vertical band appears between the body and the help line

  @slice1 @edge
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

  @slice1 @happy
  Scenario: Content windows derive from body height
    Given the terminal height increases from 30 to 40 rows at 120 columns
    When the main view re-renders
    Then more queue rows and lyric lines become visible
    And no mandatory element (title, now-playing, queue, help, visualizer) is clipped

  @slice1 @edge
  Scenario: Slim rail at narrow width
    Given a terminal narrower than 90 columns
    When the main view renders
    Then the sidebar renders as a slim rail, not stacked above nor fully hidden
    And the main area (lyrics) keeps its maximum width
    And the artwork panel is not present
    And no rendered line exceeds the terminal width

  @slice1 @happy
  Scenario: All colors match the Caelestia palette
    When the TUI renders
    Then active/nav elements use accent #e0aaff
    And secondary text uses muted #a0a0a0
    And selection uses highlight #00f5d4

  @slice2 @happy
  Scenario: No opaque background on any style including new ones
    Given every constructed style is inspected
    When title, panel, sidebar, nav-active, section-header, footer card, and the list delegate are checked
    Then none exposes any Background color
    And their rounded borders and accents remain

  @slice1 @happy
  Scenario: Core main-view elements preserved in sidebar and main
    Given a playing track with queue, lyrics, and artwork at a wide size
    When the main view renders
    Then the sidebar shows the nav and queue (window, ▲/▼ N más, ⤓, current ▶)
    And the main area shows artwork above the synced lyrics with the active line highlighted
    And title, now-playing content, status/search, help, and visualizer are all present

  @slice2 @edge
  Scenario: Mandatory elements still fit at 60x20
    Given a terminal 60 columns by 20 rows
    When the main view renders
    Then title, now-playing content, queue heading with at least 3 rows, help, and visualizer are all present
    And no rendered line exceeds 60 columns

  @slice2 @happy
  Scenario: Footer card shows now-playing content
    Given a playing track with a known position, duration, and volume
    When the main view renders
    Then a bordered accented footer card appears below the body and above the visualizer
    And it shows the ▶ state glyph, track title, progress bar, pos/dur time, and vol N

  @slice2 @edge
  Scenario: Footer card does not clip elements at 20 rows
    Given a terminal 60 columns by 20 rows with the footer card present
    When the main view renders
    Then the title, footer card, queue, help, and visualizer all remain visible
    And no rendered line exceeds 60 columns

  @slice2 @happy
  Scenario: Section headers render with an accent bar
    Given the sidebar and main area render at a wide size
    When their section headings are inspected
    Then each heading is preceded or underlined by an accent-colored bar using #e0aaff
    And no accent-bar style exposes a Background color

  @slice2 @happy
  Scenario: Active nav item is accented
    Given the default main view (queue active)
    When the sidebar nav renders
    Then the active nav item is rendered in the accent color
    And the remaining nav items use the muted color

  @slice3 @happy
  Scenario: Library renders in main with a persistent sidebar
    Given library mode is active
    When the view renders at a wide size
    Then the sidebar persists with "Biblioteca" accented
    And the library tabs, cursor ➤, and help render inside the main area
    And no delegate row applies an opaque background

  @slice3 @edge
  Scenario: Results modal and pickers stay full-screen
    Given the results modal or a lyric picker is active
    When it renders
    Then it renders full-screen (not inside the sidebar+main body)
    And its rounded borders, accents, and translucent delegate rows are preserved

  @slice3 @edge
  Scenario: 120x40 golden locks vertical fill
    Given the sidebar + main layout at 120x40
    When the view renders
    Then it matches the committed 120x40 golden fixture
    And no rendered line exceeds 120 columns
    And the fixture shows no fully-blank vertical band

  @slice3 @edge
  Scenario: Goldens differ across sizes
    Given regenerated golden fixtures
    When the 80x24, 120x30, and 120x40 goldens are compared
    Then no two of them are byte-identical
