# Design: Bootstrap TerminalTube MVP

## Technical Approach

Binario Go único. UI con Bubble Tea (modelo Elm: Model/Update/View) + Lip Gloss.
`mpv` se lanza una vez en `--idle` y se controla por socket Unix con JSON. La
búsqueda invoca `yt-dlp`. La reproducción delega la resolución de audio al hook
yt-dlp de mpv. Trabajo bloqueante (búsqueda, eventos de mpv) se modela como `tea.Cmd`.

## Architecture Decisions

### Decision: Control de mpv por IPC socket
**Choice**: Un `mpv --idle --no-video --input-ipc-server=<sock>` + cliente JSON.
**Alternatives considered**: Un proceso mpv por canción.
**Rationale**: Permite pausa/volumen/posición reales y auto-avance sin reiniciar.

### Decision: Resolución de audio vía hook yt-dlp de mpv
**Choice**: `loadfile https://youtube.com/watch?v=<id>` con `--ytdl-format=bestaudio`.
**Alternatives considered**: `yt-dlp -f ba --get-url` y pasar la URL a mpv.
**Rationale**: Evita URLs caducas y una llamada extra; mpv ya integra yt-dlp.

### Decision: Eventos de mpv → UI por canal
**Choice**: Goroutine lee el socket y emite `player.Event`; la UI los recibe con un
`tea.Cmd` que lee del canal. `tea.Tick` (1s) refresca la barra de progreso.
**Rationale**: Mantiene la UI no bloqueante y desacopla mpv de Bubble Tea.

### Decision: Persistencia en JSON (no SQLite en MVP)
**Choice**: Historial en `history.json` (XDG data dir).
**Rationale**: Fase 1 no requiere consultas; SQLite se introduce en Fase 2.

## Data Flow

    teclado ─▶ ui.Update ─▶ search.Search (cmd) ─▶ resultados ─▶ Model
                  │
                  └▶ player.Load/Pause/Volume ─▶ mpv (socket JSON)
                                                   │
    mpv end-file/pos ─▶ player.Events (chan) ─▶ ui.Update ─▶ queue.Next ─▶ Load
                  │
                  └▶ history.Add ─▶ history.json

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Create | Módulo `github.com/alexcasdev/terminaltube` |
| `main.go` | Create | Carga config, logger, valida deps, arranca programa |
| `internal/config/config.go` | Create | Viper: defaults + rutas XDG |
| `internal/logging/logging.go` | Create | Zap a archivo |
| `internal/search/search.go` | Create | `Result`, interface `Searcher` |
| `internal/search/ytdlp.go` | Create | Implementación yt-dlp (NDJSON) |
| `internal/player/player.go` | Create | Interface `Player`, `State`, `Event` |
| `internal/player/mpv.go` | Create | Proceso mpv + cliente IPC |
| `internal/queue/queue.go` | Create | `Queue`: Add/Current/Next/Prev |
| `internal/history/history.go` | Create | Append/load JSON |
| `internal/ui/*.go` | Create | model/update/view/keys/styles/messages |
| `README.md` | Create | Deps, instalación, atajos |

## Interfaces / Contracts

```go
// search
type Result struct { ID, Title, Uploader string; Duration int }
type Searcher interface { Search(ctx context.Context, q string, n int) ([]Result, error) }

// player
type State struct { Playing bool; Paused bool; Volume int; Pos, Dur float64 }
type Event struct { Kind EventKind } // EndFile, PositionUpdate, Loaded
type Player interface {
    Load(id string) error
    TogglePause() error
    SetVolume(delta int) error
    Position() (pos, dur float64)
    Events() <-chan Event
    Close() error
}

// queue
type Queue struct{ /* items + idx */ }
func (q *Queue) Add(r search.Result); func (q *Queue) Current() (search.Result, bool)
func (q *Queue) Next() bool; func (q *Queue) Prev() bool
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | queue next/prev/enqueue/bordes | table tests |
| Unit | parseo NDJSON de yt-dlp | fixtures de salida |
| Unit | history round-trip JSON | tmp dir |
| E2E | TUI: buscar→reproducir→pausa→siguiente→salir | smoke manual |

## Migration / Rollout

No migration required (greenfield).

## Open Questions

- [ ] Vista de historial como panel dedicado vs solo persistencia (MVP: solo persiste).
