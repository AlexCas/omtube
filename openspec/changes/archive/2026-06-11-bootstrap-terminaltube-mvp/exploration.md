# Exploration: bootstrap-terminaltube-mvp

## Source

Idea capturada en `idea-terminaltube-reproductor-d-2026-06-11.md`: **TerminalTube**,
reproductor de música TUI para Linux/Omarchy que usa YouTube como fuente, con
`yt-dlp` para búsqueda/resolución de audio y `mpv` como motor de reproducción.

## Repo state

- Sin código: solo orquestador SDD ARCHON (`CLAUDE.md`, `.archon/config.yaml`),
  y `openspec/` vacío. Greenfield.
- No hay `go.mod` ni paquetes previos.

## Toolchain verificado (PATH)

| Herramienta | Versión |
|-------------|---------|
| go          | 1.26.4  |
| yt-dlp      | 2026.03.17 |
| mpv         | 0.41.0  |
| sqlite3     | 3.53.1  |
| git         | 2.54.0  |

## Decisiones del usuario

- Flujo: proceso SDD/openspec formal.
- Alcance: solo Fase 1 (MVP).
- Motor mpv: control vía IPC socket (mpv único en `--idle`, comandos JSON).

## Hallazgos técnicos clave

- **Búsqueda sin descarga:** `yt-dlp "ytsearchN:query" --dump-json --flat-playlist`
  emite NDJSON con `id`, `title`, `uploader`/`channel`, `duration`.
- **mpv IPC:** `mpv --idle=yes --no-video --no-terminal --input-ipc-server=SOCK`
  acepta JSON por socket Unix: `loadfile`, `set_property pause/volume`,
  `get_property time-pos/duration`, eventos `end-file` y `observe_property`.
- **Resolución de audio:** mpv tiene hook yt-dlp integrado; pasar la URL `watch?v=ID`
  con `--ytdl-format=bestaudio` evita URLs caducas (alternativa: `yt-dlp --get-url`).
- **Patrón de referencia:** `omarchyoutube/openspec/` provee convención de artefactos.

## Reuso

- Librerías Go del ecosistema Charm: `bubbletea`, `lipgloss`, `bubbles`.
- `spf13/viper` (config), `go.uber.org/zap` (logs a archivo, no a stdout/TUI).
- Binarios de sistema reutilizados como motor: `yt-dlp`, `mpv`.
