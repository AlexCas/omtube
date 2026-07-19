Feature: Caelestia UI Visual Redesign
  Omusic's TUI is visually redesigned with rounded borders, defined
  sections, and a cohesive Caelestia color palette while preserving all
  existing keyboard shortcuts and behaviors.

  Background:
    Given Omusic is running in a terminal with at least 80 columns and 24 rows

  @happy
  Scenario: TUI renders with rounded borders at 80×24
    When the TUI renders at 80×24
    Then all panels have rounded borders
    And no panel overflows or breaks layout

  @happy
  Scenario: Queue panel shows tracks with correct highlighting
    Given multiple tracks are in the queue
    When the queue panel is displayed
    Then the current track is visually highlighted
    And cache indicators are shown per track

  @happy
  Scenario: Now playing bar shows progress and controls
    Given a track is playing
    When the now playing bar is rendered
    Then the bar shows playback state, track title, progress bar, time, and volume in one cohesive bar

  @happy
  Scenario: Lyrics panel displays synced lyrics with active line highlighted
    Given a track with synced lyrics is playing
    When the lyrics panel is visible
    Then the active line is highlighted in the Caelestia highlight color

  @happy
  Scenario: All colors match Caelestia palette after redesign
    When the TUI renders
    Then backgrounds use deep blue #1a1a2e
    And active elements use accent #e0aaff
    And secondary text uses muted #a0a0a0
    And selection uses highlight #00f5d4

  @happy
  Scenario: Artwork panel renders when available
    Given the terminal supports images and artwork is cached
    When the artwork panel is visible
    Then the artwork renders within rounded borders

  @happy
  Scenario: Queue panel shows tracks with sliding window and cache indicator
    Given the queue has more tracks than fit in the panel
    When the current track is near the end of the queue
    Then the panel scrolls to keep the current track visible
    And each track shows a cache indicator

  @edge
  Scenario: Small terminal (80×24) does not overflow or break layout
    When the terminal is exactly 80 columns by 24 rows
    Then no text overflows
    And content too long is truncated with ellipsis

  @happy
  Scenario: All existing keyboard shortcuts still work after redesign
    Given the redesigned TUI is active
    When the user presses Space, n, p, +, -, /, or q
    Then each shortcut performs its original function without change
