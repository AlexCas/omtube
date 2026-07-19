Feature: MPRIS v2 D-Bus Server
  Omusic registers as an MPRIS media player on the session bus, exposing
  metadata, playback status, position, volume, and transport controls to
  external clients like desktop widgets.

  Background:
    Given Omusic is running with the MPRIS service active

  @happy
  Scenario: Widget enumerates Omusic at startup
    Given the D-Bus session bus is available
    When Omusic starts
    Then the name org.mpris.MediaPlayer2.omusic is registered
    And external clients can enumerate Omusic as a media player

  @happy
  Scenario: Widget shows correct metadata when track changes
    Given a track with title, artist, and album is playing
    When the track changes
    Then the Metadata property contains xesam:title, xesam:artist, and xesam:album for the new track
    And when lyrics are available, xesam:asText is included

  @happy
  Scenario: Widget shows correct playback status on play/pause/stop
    Given the player is in "Playing" state
    When playback is paused
    Then PlaybackStatus changes to "Paused"
    And when playback is resumed
    Then PlaybackStatus changes to "Playing"

  @happy
  Scenario: PlayPause from widget pauses/resumes playback
    Given a track is playing
    When the PlayPause method is called via D-Bus
    Then playback toggles between Playing and Paused

  @happy
  Scenario: Next/Previous from widget advances/retreats queue
    Given the queue has multiple tracks
    When Next is called via D-Bus
    Then the queue cursor advances to the next track
    And when Previous is called via D-Bus
    Then the queue cursor moves to the previous track

  @happy
  Scenario: Seek from widget updates position
    Given a track is playing at position P µs
    When Seek is called with offset +30s (30000000 µs)
    Then the new position is approximately P + 30000000 µs
    And the Seeked signal is emitted

  @happy
  Scenario: Volume from widget updates player volume
    Given the player volume is set to 0.5
    When SetVolume is called with 0.8 via D-Bus
    Then the player volume becomes 0.8

  @edge
  Scenario: D-Bus handler dispatches via tea.Msg without touching player
    Given a D-Bus client calls PlayPause
    When the method handler executes
    Then it emits a tea.Msg into the event loop via prog.Send
    And it does not call any player or queue API directly

  @edge
  Scenario: Queue ends and PlaybackStatus becomes Stopped
    Given the last track in the queue finishes playing
    When the queue cursor reaches the end
    Then PlaybackStatus becomes "Stopped"

  @error
  Scenario: D-Bus unavailable, Omusic starts normally with warning
    Given the D-Bus session bus is not available
    When Omusic starts
    Then a warning is logged
    And the application starts and operates normally without MPRIS
