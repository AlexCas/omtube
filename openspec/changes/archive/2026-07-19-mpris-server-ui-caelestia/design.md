# Design: MPRIS Server & Caelestia UI

## Technical Approach

Add a native MPRIS v2 D-Bus server (`internal/mpris/`) following the exact nil-safe,
fire-and-forget pattern of `internal/presence/`: injected via `ui.Services`, updated
in `onTrackChange`/`clearQueue`/`EventEndFile`, and called from `main.go`.
Bidirectional control uses `prog.Send` to inject typed `tea.Msg` values into the
Bubble Tea loop — D-Bus handlers never touch player or queue directly.

The UI redesign applies a new Caelestia palette and `RoundedBorder()` to all panels
in `styles.go`, restructures `view.go` to place the now-playing bar at the top,
and splits the middle area into queue | lyrics | artwork columns — all gated on
service availability (nil panel = hidden). Existing keyboard shortcuts are
untouched.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|----------|--------|----------|-----------|
| D-Bus library | `godbus/dbus/v5` | `raitonoberu/mpris` | Standard pure-Go library; wrapper adds abstraction without benefit for this single-interface use case |
| MPRIS always active | Yes, no toggle | Config flag | User requirement: always visible to Caelestia widget |
| Bidirectional control | `prog.Send(tea.Msg)` from D-Bus handlers | Call player/queue directly | Prevents goroutine data races; matches spec "Bidirectional Isolation" requirement |
| Position property | UI pushes via `SetPosition` every second | D-Bus handler calls `player.Position()` | Keeps D-Bus goroutine isolated; 1s accuracy is acceptable for polling widget |
| Lyrics in metadata | `SetMetadata` called on track change AND when lyrics arrive | Single call with async await | Lyrics resolve asynchronously; two-step update ensures metadata is correct as soon as available |
| Name on D-Bus | `org.mpris.MediaPlayer2.omusic` | — | Unique, follows MPRIS convention |
| MPRIS construction timing | After `tea.NewProgram`, injected via setter | In `buildServices` | MPRIS needs `prog.Send` which exists only after `prog` creation |

## Data Flow

```
D-Bus client (Caelestia widget)
  │
  │ PlayPause / Next / Seek / SetVolume
  ▼
MPRIS server (goroutine)
  │ prog.Send(mpris.PlayPauseMsg{})
  ▼
Model.Update (main goroutine)
  │ m.player.TogglePause()
  │ m.mpris.SetPlaybackStatus("Playing")
  │ m.mpris.SetVolume(vol)
  ▼
MPRIS server (state updated, PropertiesChanged emitted)

UI → MPRIS state push:
  Model (tickMsg/posMsg handler)
    │ m.mpris.SetPosition(pos)
    ▼
  MPRIS server (sync.Mutex → stored for D-Bus Get)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/mpris/server.go` | Create | D-Bus server: registration, property handlers, metadata mapping, state mutex, `prog.Send` dispatch |
| `internal/mpris/server_test.go` | Create | Unit tests for metadata mapping and message dispatch |
| `internal/ui/styles.go` | Modify | Caelestia palette (#1a1a2e/#e0aaff/#a0a0a0/#00f5d4), `RoundedBorder()` on all panels |
| `internal/ui/view.go` | Modify | Restructure layout: now-playing at top, queue/lyrics/artwork in horizontal panels, help at bottom |
| `internal/ui/update.go` | Modify | Handle `mpris.*Msg` types (play/pause/next/prev/stop/seek/volume); push state to `mprisService` on track change, end-of-queue, lyrics arrival, position tick |
| `internal/ui/model.go` | Modify | Add `mprisService` interface, `mpris` field in `Model`, `Mpris` in `Services`; expose `SetMpris()` setter |
| `main.go` | Modify | Create `mpris.New(prog.Send, logger)` after `prog`, inject via `model.SetMpris()`, defer `Close()` |
| `go.mod` | Modify | Add `github.com/godbus/dbus/v5` |

## Interfaces / Contracts

**`mprisService` interface** (`internal/ui/model.go` — follows `presenceService` pattern):

```go
type mprisService interface {
    SetMetadata(track search.Result, lyrics lyrics.Lyrics)
    SetPlaybackStatus(status string) // Playing | Paused | Stopped
    SetVolume(vol int)               // 0–130
    SetPosition(pos float64)         // seconds
    Seeked(offsetUS int64)           // emit Seeked signal
    Close()
}
```

**`tea.Msg` types for D-Bus → UI commands** (`internal/mpris/server.go`):

```go
type PlayPauseMsg struct{}
type NextMsg      struct{}
type PrevMsg      struct{}
type StopMsg      struct{}
type SeekMsg      struct{ Offset int64 }   // µs
type SetVolumeMsg struct{ Volume float64 } // 0.0–1.0
```

**`Services` struct** (add to `internal/ui/model.go`):
```go
type Services struct {
    // ... existing fields ...
    Mpris mprisService  // nil ⇒ feature unavailable
}
```

**Metadata mapping** (`search.Result` → D-Bus dict, in `internal/mpris/server.go`):

```go
func metadataDict(track search.Result, lyrics lyrics.Lyrics) map[string]dbus.Variant {
    m := map[string]dbus.Variant{
        "xesam:title":  dbus.MakeVariant(track.Title),
        "xesam:artist": dbus.MakeVariant(track.Uploader),
        "xesam:album":  dbus.MakeVariant(""),
        "mpris:length": dbus.MakeVariant(int64(track.Duration) * 1e6),
        "mpris:artUrl": dbus.MakeVariant("https://i.ytimg.com/vi/" + track.ID + "/hqdefault.jpg"),
    }
    if lyrics.Synced {
        lines := make([]string, len(lyrics.Lines))
        for i, l := range lyrics.Lines {
            lines[i] = l.Text
        }
        m["xesam:asText"] = dbus.MakeVariant(lines)
    }
    return m
}
```

**MPRIS server constructor** (`internal/mpris/server.go`):
```go
func New(send func(msg interface{}), logger *zap.Logger) (*Server, error)
// Returns nil + error when D-Bus session bus is unavailable.
// All methods are nil-safe (guard with `if s == nil { return }`).
```

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | Metadata mapping (`search.Result` → D-Bus dict) | Table-driven test in `server_test.go`; verify title, artist, length (µs conversion), artUrl, asText presence |
| Unit | PlayPause/Next/Prev message dispatch | Mock `send` callback captures emitted `tea.Msg` type; verify correct msg on each D-Bus method call |
| Unit | Volume conversion (MPRIS 0.0–1.0 ↔ player 0–130) | Round-trip test with edge cases (0, 0.5, 1.0) |
| Unit | UI message handlers (update_test.go) | Extend existing fakePlayer test suite: send `mpris.PlayPauseMsg{}` and verify `fakePlayer.paused` toggles |
| Integration | D-Bus without real session bus | Use `godbus/dbus/v5` test helpers or stub the `dbus.Conn` — skip if unavailable, tested manually against real bus |
| Visual | Layout at 80×24 and 120×30 | `go test` with snapshot/golden file comparison on `View()` output |

## Migration / Rollback

**No data migration.** Rollback: delete `internal/mpris/`, revert `styles.go`/`view.go`/`update.go`/`model.go`/`main.go`, remove `godbus/dbus/v5` from `go.mod` and run `go mod tidy`.

## Open Questions

- None. All technical decisions are resolved from the specs and codebase analysis.
