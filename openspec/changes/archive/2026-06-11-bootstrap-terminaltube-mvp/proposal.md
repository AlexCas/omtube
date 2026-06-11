# Proposal: Bootstrap TerminalTube MVP

## Intent

No existe reproductor de música ligero, controlado por teclado, para usar dentro de
la terminal en Linux/Omarchy. TerminalTube cubre esa necesidad usando YouTube como
fuente vía `yt-dlp` y `mpv`, sin depender de APIs oficiales ni de un navegador.

## Scope

### In Scope
- Búsqueda de canciones en YouTube y listado de resultados en la TUI.
- Reproducción de audio controlando `mpv` por IPC socket.
- Cola de reproducción con avance/retroceso y auto-avance al terminar una pista.
- Atajos de teclado (play/pausa, siguiente, anterior, volumen, buscar, salir).
- Historial local de reproducciones en archivo JSON.
- Configuración por archivo (Viper) y logging a archivo (Zap).

### Out of Scope
- SQLite, playlists, favoritos persistentes (Fase 2).
- Letras, portadas, caché de descargas, Discord Rich Presence (Fase 3).
- Integración oficial con YouTube Music / Premium.

## Capabilities

### New Capabilities
- `youtube-search`: buscar en YouTube vía yt-dlp y exponer resultados estructurados.
- `audio-playback`: controlar mpv (load/pausa/volumen/posición/fin) por IPC.
- `playback-queue`: gestionar la cola y la pista actual.
- `tui-shell`: interfaz Bubble Tea con búsqueda, resultados, cola y atajos.
- `playback-history`: registrar lo reproducido en JSON local.

## Approach

App Go (binario único) con UI Bubble Tea + Lip Gloss. `mpv` se lanza una vez en
`--idle` y se controla por socket Unix con comandos JSON; eventos de mpv (`end-file`,
posición) se propagan a la UI por un canal y `tea.Cmd`. La búsqueda usa
`yt-dlp ytsearchN: --dump-json --flat-playlist`. Config con Viper en rutas XDG.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod`, `main.go` | New | Módulo y entrypoint |
| `internal/{search,player,queue,history}` | New | Lógica de dominio |
| `internal/ui` | New | TUI Bubble Tea |
| `internal/{config,logging}` | New | Config Viper + logs Zap |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Cambios de YouTube rompen yt-dlp | Med | Aislar tras interface `Searcher`; mensaje de error claro; yt-dlp actualizable |
| Falta yt-dlp/mpv en PATH | Med | Validar al arranque y abortar con mensaje accionable |
| Latencia de resolución de audio | Low | Resolver en goroutine; estado "cargando" en UI |

## Rollback Plan

Greenfield: revertir = eliminar el módulo Go y el change folder. No afecta nada
existente (no había código).

## Dependencies
- Binarios en PATH: `yt-dlp`, `mpv`.
- Libs Go: charmbracelet/{bubbletea,lipgloss,bubbles}, spf13/viper, go.uber.org/zap.

## Success Criteria
- [ ] `go build ./...` y `go vet ./...` sin errores.
- [ ] Buscar una canción y reproducirla produce audio vía mpv.
- [ ] Cola avanza sola al terminar; atajos play/pausa/volumen/siguiente funcionan.
- [ ] El historial queda persistido en JSON.
- [ ] Tests unitarios de queue/search/history en verde.
