# Tasks: MPRIS Server & Caelestia UI

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–700 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: MPRIS server + wiring — PR 2: UI redesign |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | MPRIS server (`internal/mpris/`, `Main` wiring, `mprisService` interface, `go.mod`) | PR 1 | Base on main; self-contained — isolated package, tests pass w/o UI changes |
| 2 | Caelestia UI redesign (`styles.go`, `view.go`, `update.go` MPRIS hooks, `messages.go`) | PR 2 | Base on main (or PR 1 if stacked); pure visual + wiring, no MPRIS pkg changes |

## Phase 1: Foundation

- [x] 1.1 Add `github.com/godbus/dbus/v5` to `go.mod` and run `go mod tidy`
- [x] 1.2 Define `mprisService` interface (`SetMetadata`, `SetPlaybackStatus`, `SetVolume`, `SetPosition(pos)`, `Seeked`, `Close`) in `internal/ui/model.go`
- [x] 1.3 Add `Mpris mprisService` field to `Services` struct and `SetMpris()` setter in model

## Phase 2: MPRIS Server Core

- [x] 2.1 Create `internal/mpris/server.go` with `New(send, logger)` constructor and D-Bus registration on `org.mpris.MediaPlayer2.omusic`
- [x] 2.2 Implement `Metadata`, `PlaybackStatus`, `Position`, `Volume` D-Bus properties with `sync.Mutex`-guarded state
- [x] 2.3 Implement `PlayPause`, `Next`, `Previous`, `Stop`, `Seek`, `SetVolume` D-Bus methods dispatching `tea.Msg` via `prog.Send`
- [x] 2.4 Implement `metadataDict()` mapping `search.Result` + `lyrics.Lyrics` to D-Bus dict (xesam:title, artist, album, mpris:length/artUrl, xesam:asText)
- [x] 2.5 Define `PlayPauseMsg`, `NextMsg`, `PrevMsg`, `StopMsg`, `SeekMsg`, `SetVolumeMsg` types in `internal/mpris/server.go`
- [x] 2.6 Implement nil-safe `Close()` releasing D-Bus name and connection

## Phase 3: UI Integration

- [x] 3.1 Handle `mpris.*Msg` types in `internal/ui/update.go` — dispatch to player/queue methods (PlayPause toggles, Next/Prev advance queue, Seek seeks, SetVolume adjusts, Stop stops)
- [x] 3.2 Push MPRIS state from `onTrackChange` handler: call `m.SetMetadata()` and `m.SetPlaybackStatus("Playing")`
- [x] 3.3 Push MPRIS state on end-of-queue (`EventEndFile`): call `m.SetPlaybackStatus("Stopped")`
- [x] 3.4 Push MPRIS state on lyrics arrival: call `m.SetMetadata(track, lyrics)` to update `xesam:asText`
- [x] 3.5 Push MPRIS position from `posMsg` tick: call `m.SetPosition(pos)` every second
- [x] 3.6 Wire MPRIS in `main.go`: create `mpris.New(prog.Send, logger)` after `tea.NewProgram`, inject via `model.SetMpris()`, defer `Close()`

## Phase 4: UI Redesign — Caelestia

- [x] 4.1 Update `internal/ui/styles.go` with Caelestia palette (`#1a1a2e` background, `#e0aaff` accent, `#a0a0a0` muted, `#00f5d4` highlight)
- [x] 4.2 Apply `lipgloss.RoundedBorder()` to all panel styles in `styles.go`
- [x] 4.3 Restructure `internal/ui/view.go`: now-playing bar at top, queue/lyrics/artwork in horizontal middle section, help at bottom
- [x] 4.4 Add nil-guards in `view.go` so nil service panels are hidden gracefully

## Phase 5: Testing

- [x] 5.1 Write table-driven tests in `internal/mpris/server_test.go` for `metadataDict()` — verify title, artist, length (µs), artUrl, asText
- [x] 5.2 Test message dispatch: mock `send` callback, call each D-Bus method, verify correct `tea.Msg` type emitted
- [x] 5.3 Test volume round-trip conversion (0.0–1.0 ↔ 0–130) with edge cases
- [x] 5.4 Extend `update_test.go` with fake player: send `mpris.PlayPauseMsg{}` and verify pause toggles
- [x] 5.5 Add golden-file/snapshot test for `View()` at 80×24 and 120×30

## Phase 6: Cleanup

- [x] 6.1 Run `go mod tidy` to pin `godbus/dbus/v5` and clean `go.sum`
- [x] 6.2 Add package-level doc comments to `internal/mpris/` exported symbols
