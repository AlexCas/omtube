# Tasks: Bootstrap TerminalTube MVP

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~900–1200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1 scaffold+core, PR2 player(mpv), PR3 TUI+wiring |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Scaffold + config/logging + search/queue/history | PR 1 | base sin mpv ni UI |
| 2 | Player mpv por IPC | PR 2 | depende de Unit 1 |
| 3 | TUI Bubble Tea + wiring | PR 3 | depende de Units 1–2 |

> Nota: el usuario aprobó entrega MVP en un solo paso (size-exception). Se implementa
> completo; los work units guían el orden, no PRs separados.

## Phase 1: Scaffold & Foundation

- [x] 1.1 `go mod init github.com/alexcasdev/terminaltube`; añadir deps (bubbletea, lipgloss, bubbles, viper, zap)
- [x] 1.2 `internal/config/config.go`: Viper, defaults (results=10, volume=70), rutas XDG
- [x] 1.3 `internal/logging/logging.go`: Zap a archivo en XDG state/cache
- [x] 1.4 `main.go`: cargar config/logger, validar yt-dlp y mpv en PATH, arrancar programa

## Phase 2: Domain Core

- [x] 2.1 `internal/search/search.go`: `Result`, interface `Searcher`
- [x] 2.2 `internal/search/ytdlp.go`: ejecutar yt-dlp, parsear NDJSON, descartar sin id
- [x] 2.3 `internal/queue/queue.go`: Add/Current/Next/Prev (sin wrap)
- [x] 2.4 `internal/history/history.go`: Add + load/save JSON en XDG data dir

## Phase 3: Player (mpv IPC)

- [x] 3.1 `internal/player/player.go`: interfaces `Player`, `State`, `Event`/`EventKind`
- [x] 3.2 `internal/player/mpv.go`: lanzar mpv idle + socket; loadfile; set_property pause/volume
- [x] 3.3 mpv.go: goroutine lectora del socket → eventos end-file y posición por canal; Close limpio

## Phase 4: TUI

- [x] 4.1 `internal/ui/keys.go` + `styles.go`: key.Binding y estilos Lip Gloss
- [x] 4.2 `internal/ui/messages.go` + `model.go`: tipos msg, Model, Init
- [x] 4.3 `internal/ui/update.go`: teclas (/,espacio,n,p,+,-,q), msgs (resultados, tick, evento mpv), auto-avance, history.Add
- [x] 4.4 `internal/ui/view.go`: paneles búsqueda/resultados/cola + barra de estado/progreso

## Phase 5: Docs & Verify

- [x] 5.1 `README.md`: deps (yt-dlp, mpv), build/run, atajos
- [x] 5.2 Tests: queue, search (fixtures), history (tmp dir)
- [x] 5.3 `go build ./...`, `go vet ./...`, `go test ./...`; smoke test; `verify-report.md`
