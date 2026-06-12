# Archive Report: add-media-enrichment (Fase 3)

**Archived**: 2026-06-12
**Mode**: openspec
**Branch**: feature/media-enrichment
**SDD cycle**: explore → propose → spec → design → tasks → apply → verify → judge → archive (todas completadas)

## Change Summary

Fase 3 añade la capa de enriquecimiento multimedia sobre la biblioteca persistida de Fase 2.
Cuatro capacidades nuevas y dos modificadas dan una experiencia más rica: las pistas repetidas
cargan al instante desde una caché local, se muestra la letra (resaltando la línea sincronizada
según la posición de reproducción), se renderiza la portada en el terminal con degradación
elegante, y se publica la pista en reproducción como presencia "escuchando" en Discord. Cada
feature es opcional y desactivable por config (toggles `cache`/`lyrics`/`artwork`/`presence`),
y con todos los toggles apagados el comportamiento es idéntico al de Fase 2.

Implementación: nuevos paquetes `internal/cache` (descarga vía yt-dlp a XDG cache + índice en
SQLite con expiración por tamaño/antigüedad), `internal/lyrics` (cliente HTTP a lrclib + parser
`.lrc` con búsqueda binaria por tiempo), `internal/artwork` (detección de protocolo gráfico
kitty→sixel→chafa→none) e `internal/presence` (Discord IPC vía `rich-go`, puro Go). Migración 2
de storage (`cache_entries`, `lyrics_cache`, `user_version` 1→2). El player emite
`EventTrackChange`; `ui.Update` hace fan-out de `tea.Cmd`s (lyrics/artwork/presence/cache-download)
sin bus de goroutines ni acoplamiento entre paquetes. El binario sigue siendo puro Go / no-cgo.

## 3-PR Chain Delivery

Entregado como cadena ordenada de 3 work units (feature-branch-chain, presupuesto >400 líneas,
forecast High), cada unidad compilando y en verde antes de la siguiente:

| Unit | PR | Goal | Tasks |
|------|----|------|-------|
| 1 | PR 1 | Storage migration 2 + `internal/cache` (index, download, eviction, startup sweep) + unit tests | 1.1–1.6, 4.3, 4.6 |
| 2 | PR 2 | `internal/lyrics` + `internal/artwork` + `internal/presence` + dep `rich-go` + unit tests | 2.1–2.5, 4.1, 4.2, 4.4, 4.5 |
| 3 | PR 3 | `internal/player` eventos + cache-aware Load, `internal/config` toggles, `internal/ui` paneles/wiring, `main.go`, tests model-level, docs | 3.1–3.8, 4.7, 5.1, 5.2 |

Cada PR construye y `go test ./...` en verde de forma independiente. PR #1 apunta a la rama
tracker; #2 apunta a #1; #3 apunta a #2.

## Task Completion Gate

- `tasks.md`: **28/28** tareas marcadas `[x]`, 0 sin marcar. ✅
- `verify-report.md`: **0 issues CRITICAL** (en la verificación final post-judge). ✅

## Verify Result

**PASS WITH WARNINGS** (0 CRITICAL).

- `gofmt -l .` → vacío (sin archivos sin formatear).
- `go build ./...`, `go vet ./...` → exit 0 (limpios).
- Binario estático / no-cgo (criterio de éxito del proposal): `CGO_ENABLED=0 go build .` →
  ELF "statically linked", `ldd` → "not a dynamic executable". `rich-go` no añade cgo. ✅
- `go test ./... -count=1` → **84/84 PASS / 0 FAIL / 0 SKIP** en la verificación inicial; tras
  los fixes de judge (C1 + W1/W2/W4) subió a **89/89 PASS / 0 FAIL / 0 SKIP**.
- Spec compliance: 24 ✅ COMPLIANT / 7 ⚠️ PARTIAL / 0 ❌ UNTESTED / 0 ❌ NON-COMPLIANT de 31.
  Los PARTIAL se dividían en I/O del mundo real no ejercitable en unit tests (kitty/sixel pixel
  render, IPC real de Discord, subproceso mpv) más dos gaps de comportamiento (W1, W2) que el
  judge marcó para corrección.

## Judge Result

**APPROVED — round 2** (judgment-day dual review).

- **Round 1: ISSUES FOUND** — 1 CRITICAL (C1) + 4 WARNING (W1, W2, W3, W4) confirmados.
- Fixes aplicados y re-verificados (código real + tests que afirman el comportamiento); el
  conteo de tests pasó de 84 a 89:
  - **C1 (CRITICAL)** — corregido y cubierto con test; re-verify confirma 0 CRITICAL y suite en
    verde (89/89). El arreglo cierra el bloqueo que impedía aprobar el ciclo.
  - **W1 — render kitty/sixel era placeholder, no gráficos reales.** Corregido: `Backend.Render`
    para kitty/sixel ahora emite las secuencias de escape del protocolo soportado en vez de
    devolver `[sin portada]`; la degradación a `chafa`/placeholder se mantiene para terminales
    sin soporte. (El render nativo en pantalla real sigue siendo verificable solo en TTY real;
    ver Known Issues.)
  - **W2 — presencia no se limpiaba al detener la reproducción.** Corregido: al terminar la cola
    (`update.go`, queue-finished) ahora se invoca `presence.Clear()`, no solo al salir
    (`defer Close()`→`logout`). Test afirma que `Clear()` se llama en el camino de stop.
  - **W4 — decisión "reusar thumbnail cacheado" no implementada.** Corregido: el thumbnail
    escrito por `--write-thumbnail` ahora se indexa y `artworkAdapter.Render` prefiere el
    archivo local cacheado antes de recurrir a la URL remota `i.ytimg.com/.../hqdefault.jpg`,
    eliminando la descarga remota redundante en pistas totalmente cacheadas y habilitando
    portada offline.
  - **W3 — deuda de tests teatest (golden frames).** NO corregido: aceptado y diferido a fase
    futura (ver Known Issues). El UI de Fase 3 sí quedó cubierto a nivel de modelo (9 funcs).
- **Round 2: re-verify** build/vet/gofmt/test en verde (89/89), sin regresiones → **APPROVED**
  por ambos jueces.

## Specs Synced

Las delta del change se fusionaron en las specs maestras de `openspec/specs/`:

| Domain | Created/Updated | Requirements |
|--------|-----------------|--------------|
| download-cache | Created (nueva capability) | 3 (Local Audio Cache, Cache Lookup Priority, Cache Eviction) |
| lyrics | Created (nueva capability) | 3 (Fetch Lyrics, Lyrics Unavailable, Synced Line Highlight) |
| artwork | Created (nueva capability) | 2 (Terminal Graphics Detection, Render Current Track Artwork) |
| discord-rich-presence | Created (nueva capability) | 2 (Presence Connection, Publish Now Playing) |
| audio-playback | Updated (2 MODIFIED) | 3 (Single mpv via IPC [preservado] + Load and Transport Control [modificado], Progress and End Events [modificado]) |
| tui-shell | Updated (3 ADDED) | 8 (5 previos preservados + Lyrics Panel, Artwork Panel, Cache Indicator) |

### Reconciliación de audio-playback (2 MODIFIED, sin huérfanos)

Ambos requirements MODIFIED del delta **reemplazaron el bloque completo** del requirement
homónimo en la spec maestra, preservando todos los escenarios:

- **"Load and Transport Control"**: el bloque previo (que cargaba siempre por id de YouTube) se
  reemplazó por la versión que prioriza el archivo cacheado válido. El escenario previo "Play a
  track" se preservó (ahora condicionado a "no cacheada"), se conservó "Toggle pause and volume",
  y se añadió el escenario nuevo "Play a cached track". Verificado: `grep "load a track by
  YouTube id"` en `openspec/specs/` → sin coincidencias (texto antiguo eliminado); un solo
  heading "### Requirement: Load and Transport Control".
- **"Progress and End Events"**: el bloque se reemplazó para añadir la emisión del evento de
  cambio de pista. El escenario "Track ends" se preservó intacto y se añadió "Track changes".
  Un solo heading "### Requirement: Progress and End Events".
- El requirement **"Single mpv via IPC"** (no mencionado en el delta) se preservó intacto.

### Reconciliación de tui-shell (3 ADDED)

Los tres requirements nuevos (**Lyrics Panel**, **Artwork Panel**, **Cache Indicator**) se
añadieron al final de la sección de requirements. Los 5 requirements previos (Search Input Mode,
Results and Queue Panels, Keyboard Controls, Non-blocking UI, Library Mode, Create Playlist from
UI, Library Action Shortcuts — los de Fase 1 y Fase 2) se preservaron intactos. Verificado: cada
heading nuevo aparece exactamente 1 vez, sin duplicados.

## Known Issues (no bloqueantes, anotados — deuda arrastrada)

- **WARNING arrastrado — deuda de tests UI teatest (W3)**: `internal/ui` tiene ahora tests
  sólidos a nivel de modelo (9 funcs que afirman paneles letra/portada, fan-out en track-change,
  paridad con toggles apagados y descarte de respuestas obsoletas) — una mejora real sobre el 0%
  de Fase 2 — pero aún NO tiene cobertura de golden frames `teatest` de `Update`/`View` contra un
  TTY simulado. Es la misma deuda que Fase 2 arrastró; Fase 3 la reduce pero no la cierra.
  Diferida a fase futura. Documentada en `tasks.md` 4.7.
- **Render nativo kitty/sixel — trabajo futuro / verificación TTY**: tras el fix de W1 el código
  emite secuencias de escape del protocolo gráfico, pero el render real en pantalla (pixeles
  kitty/sixel, IPC real de Discord, subproceso mpv) solo es verificable en un terminal real y no
  está cubierto por unit tests. Recomendado un smoke manual en TTY real (S2 del verify-report) y
  considerar el endurecimiento del encoding nativo kitty/sixel como trabajo futuro.
- **SUGGESTION S1 (no bloqueante)**: `internal/player`, `internal/config` y `main` no tienen
  unit tests directos (verificados vía tests model-level del UI + build + razonamiento). Un
  `config_test.go` (tabla de verdad de `PresenceActive`) y un test de emisión de eventos del
  player endurecerían el wiring de forma barata.

## Outcome

Fase 3 (enriquecimiento multimedia: caché, letras, portadas, Discord Rich Presence) entregada en
cadena de 3 PRs, verificada (PASS WITH WARNINGS, 0 CRITICAL), juzgada (APPROVED round 2 tras
corregir C1 + W1/W2/W4; W3 diferida) y archivada. 28/28 tareas completas, suite final 89/89 en
verde, binario puro Go / no-cgo preservado. Source of truth actualizado en `openspec/specs/`: 4
capabilities nuevas (download-cache, lyrics, artwork, discord-rich-presence) y 2 modificadas
(audio-playback, tui-shell), sin requirements huérfanos ni duplicados. Deuda vigente: golden
frames `teatest` (W3) y render nativo kitty/sixel en TTY real, ambos diferidos a fase futura.
