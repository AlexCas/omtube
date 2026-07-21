Feature: Caelestia UI Responsive Redesign
  Omusic's TUI is redesigned to be translucent by default and responsive across
  width and height, preserving every existing element, keybinding, and behavior.
  Delivered in three chained slices; scenarios are tagged @slice1, @slice2, @slice3.

  Background:
    Given Omusic is running with a playing track, a queue, lyrics, and artwork

  @slice1 @happy
  Scenario: All colors match Caelestia palette after redesign
    When the TUI renders
    Then active elements use accent #e0aaff
    And secondary text uses muted #a0a0a0
    And selection uses highlight #00f5d4

  @slice1 @happy
  Scenario: No opaque background paints over the terminal glass
    Given the default styles are constructed
    When the title and panel styles are inspected
    Then neither style exposes any Background color
    And their rounded borders remain

  @slice1 @edge
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

  @slice1 @happy
  Scenario: Widths derive from runtime dimensions
    Given the terminal width changes from 80 to 120 columns
    When the main view re-renders
    Then queue, lyrics, and artwork widths and truncations change accordingly

  @slice2 @happy
  Scenario: Vertical space is used without clipping
    Given a terminal 120 columns by 30 rows
    When the main view renders
    Then content is placed using the available height
    And no mandatory element (title, now-playing, queue, help, visualizer) is clipped

  @slice2 @edge
  Scenario: Narrow breakpoint hides artwork
    Given a terminal narrower than the narrow breakpoint
    When the main view renders
    Then the artwork panel is not present
    And the queue and lyrics panels are still rendered

  @slice2 @happy
  Scenario: Breakpoints render distinct deterministic layouts
    Given identical model state
    When the view renders at 60, 80, and 120 columns
    Then the three outputs differ from each other
    And each output is deterministic for its size

  @slice1 @happy
  Scenario: Core main-view elements preserved
    Given a playing track with queue, lyrics, and artwork
    When the main view renders at a wide size
    Then title, now-playing bar, status/search, queue, lyrics, artwork, help, and visualizer are all present

  @slice2 @happy
  Scenario: Queue and lyrics behaviors preserved
    Given a queue longer than the visible window and synced lyrics
    When the panels render
    Then the queue shows the sliding window with ▲/▼ N más, cache mark ⤓, and current ▶
    And the lyrics panel highlights the active line with the highlight color

  @slice3 @happy
  Scenario: Modals, library, and pickers preserved and translucent
    Given the results modal, library view, or a lyric picker is active
    When it renders
    Then its tabs, cursor, and controls are present
    And no delegate row applies an opaque background
    And rounded borders and accents remain

  @slice1 @edge
  Scenario: 80×24 and 120×30 goldens differ
    Given regenerated golden fixtures
    When the 80×24 and 120×30 goldens are compared
    Then they are not byte-identical

  @slice2 @edge
  Scenario: Narrow 60×20 golden is locked
    Given the narrow breakpoint layout
    When the view renders at 60×20
    Then it matches the committed 60×20 golden fixture
    And no rendered line exceeds 60 columns
