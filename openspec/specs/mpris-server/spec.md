# MPRIS Server Specification

## Purpose

Exponer Omusic como reproductor MPRIS v2 en el bus de sesión D-Bus para que
widgets como el de Caelestia Quickshell puedan mostrar metadatos, progreso y
enviar controles de transporte.

## Requirements

### Requirement: MPRIS Registration

The system MUST register `org.mpris.MediaPlayer2.omusic` on the session bus at
startup. It MUST implement `org.mpris.MediaPlayer2` (desktop entry) and
`org.mpris.MediaPlayer2.Player` (playback control) interfaces.

#### Scenario: Widget enumerates Omusic at startup

### Requirement: Metadata Exposure

The system MUST expose the `Metadata` property as a D-Bus dict with
`xesam:title`, `xesam:artist`, `xesam:album`, `mpris:length` (µs),
`mpris:artUrl`. It MUST include `xesam:asText` when synced lyrics are
available. Metadata MUST update on every track change.

#### Scenario: Widget shows correct metadata when track changes

### Requirement: PlaybackStatus Exposure

The system MUST expose `PlaybackStatus` as `Playing`, `Paused`, or `Stopped`
and update it on every state change. It MUST transition to `Stopped` when the
queue ends.

#### Scenario: Widget shows correct playback status on play/pause/stop

### Requirement: Position and Volume

The system MUST expose `Position` (µs) via a pollable property that returns
live playback position. `Volume` MUST be exposed as a double in [0.0, 1.0].
`Seeked` signal MUST be emitted after every seek operation.

#### Scenario: Seek from widget updates position

#### Scenario: Volume from widget updates player volume

### Requirement: Transport Controls

The system MUST implement `PlayPause`, `Next`, `Previous`, `Stop`, `Seek`
(offset in µs), and `SetVolume` (double). `PlayPause` MUST toggle between
play and pause.

#### Scenario: PlayPause from widget pauses/resumes playback

#### Scenario: Next/Previous from widget advances/retreats queue

### Requirement: Bidirectional Isolation

D-Bus method handlers MUST NOT call player or queue APIs directly. They MUST
emit `tea.Msg` into the Bubble Tea event loop via `prog.Send`. All state
mutations MUST happen in the main UI goroutine.

#### Scenario: D-Bus handler dispatches via tea.Msg without touching player

### Requirement: Lifecycle Management

The system MUST release the D-Bus name and close the connection when the
application shuts down. It MUST NOT leak listeners.

#### Scenario: Queue ends and PlaybackStatus becomes Stopped

### Requirement: Graceful Degradation

If the D-Bus session bus is unavailable at startup, the system MUST log a
warning once and MUST continue operating normally. It MUST NOT crash or
panic.

#### Scenario: D-Bus unavailable, Omusic starts normally with warning
