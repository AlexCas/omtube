# Proposal: MPRIS Server & Caelestia UI

## Intent

Omusic is invisible to the Caelestia launcher's Quickshell MPRIS widget — no D-Bus player to enumerate, so metadata, controls, and progress are unavailable there. The current TUI also lacks the cohesive visual identity (rounded borders, defined sections) for a recognizable player. This change adds a native MPRIS server and a Caelestia-style visual redesign.

## Scope

### In Scope
- MPRIS v2 server on the session bus (`org.mpris.MediaPlayer2.omusic`), always active.
- Properties: `Metadata`, `PlaybackStatus`, `Position`, `Volume`, `LoopStatus`, `Rate`.
- Controls: `Play`, `Pause`, `PlayPause`, `Next`, `Previous`, `Stop`, `Seek`, `SetVolume`.
- Bidirectional control: D-Bus handlers emit `tea.Msg` via `prog.Send`.
- Lyrics in `xesam:asText` when available.
- UI: rounded borders, defined sections, refined palette in `styles.go`; restructured `view.go`.

### Out of Scope
- `OpenUri` (deferred pending widget support).
- MPRIS config toggle (always on).
- Playlist control beyond Next/Previous.
- Visualizer / artwork algorithm changes.

## Capabilities

### New Capabilities
- `mpris-server`: D-Bus MPRIS v2 service exposing metadata and transport controls.
- `caelestia-ui`: Visual redesign with rounded borders, defined sections, cohesive palette.

### Modified Capabilities
- None (layout changes live under `caelestia-ui`; playback stays in existing specs).

## Approach

Mirror the `internal/presence/` pattern: nil-safe, fire-and-forget service injected via `ui.Services`, hooked in `onTrackChange`, `clearQueue`, `EventEndFile`. Use `github.com/godbus/dbus/v5` to own `org.mpris.MediaPlayer2.omusic` and implement `org.mpris.MediaPlayer2` + `.Player`. D-Bus handler goroutines never touch player/queue; they emit `mprisPlayPauseMsg`, `mprisNextMsg`, `mprisPrevMsg`, `mprisStopMsg`, `mprisSeekMsg` via `prog.Send`. UI: rewrite `styles.go` for rounded borders + palette, restructure `view.go` for clear hierarchy reusing queue/lyrics/artwork/progress/visualizer.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/mpris/` | New | D-Bus server, metadata mapping, bubbletea dispatcher |
| `internal/ui/styles.go` | Modified | Caelestia palette, rounded borders |
| `internal/ui/view.go` | Modified | Restructured panel layout |
| `internal/ui/update.go` | Modified | Handle `mpris*Msg` messages |
| `internal/ui/model.go` | Modified | Service wiring + msg types |
| `main.go` | Modified | Construct mpris, inject `ui.Services` |
| `go.mod` | Modified | Add `github.com/godbus/dbus/v5` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| D-Bus name collision | Med | Unique name `org.mpris.MediaPlayer2.omusic` |
| Goroutine data races | Med | Handlers only emit `tea.Msg` |
| UI breaks on small terminal | Med | Test 80×24 + 120×30; graceful truncation |
| Listener leak on exit | Low | `Close()` in shutdown path |

## Rollback Plan

Remove `internal/mpris/`, revert `styles.go`/`view.go`/`update.go`/`model.go`, drop mpris from `main.go`, remove `godbus/dbus/v5` from `go.mod`. No migration.

## Dependencies

- `github.com/godbus/dbus/v5`; existing `internal/player`, `internal/queue`, `internal/ui.Services`.

## Success Criteria

- [ ] Caelestia widget enumerates and selects `omusic`.
- [ ] Widget shows title/artist/art/progress with a working progress bar.
- [ ] PlayPause/Next/Previous/Stop/Seek/Volume from widget control Omusic.
- [ ] TUI renders rounded, defined panels at 80×24+.
- [ ] No regressions in existing TUI keyboard control.