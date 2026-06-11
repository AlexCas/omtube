# TerminalTube

Reproductor de música TUI para Linux/Omarchy que usa YouTube como fuente, vía
[`yt-dlp`](https://github.com/yt-dlp/yt-dlp) para buscar/resolver audio y
[`mpv`](https://mpv.io) como motor de reproducción.

> Proyecto de uso personal. No es un cliente oficial de YouTube Music / Premium.

## Requisitos

- Go 1.24+
- `yt-dlp` y `mpv` disponibles en el `PATH`

En Arch/Omarchy:

```bash
sudo pacman -S mpv yt-dlp
```

## Instalación

```bash
go build -o terminaltube .
# o instalar en $GOBIN:
go install .
```

## Uso

```bash
./terminaltube
```

1. Pulsa `/` para buscar, escribe el nombre de la canción y `Enter`.
2. Mueve el cursor con `↑/↓` (o `k/j`) y `Enter` para encolar/reproducir.

### Atajos

| Tecla     | Acción          |
|-----------|-----------------|
| `/`       | Buscar          |
| `Enter`   | Encolar / reproducir |
| `Espacio` | Play / Pausa    |
| `n`       | Siguiente       |
| `p`       | Anterior        |
| `+` / `-` | Volumen ± 5     |
| `f`       | Marcar / desmarcar favorito de la pista seleccionada |
| `a`       | Añadir la pista seleccionada a una playlist |
| `L`       | Abrir / cerrar la biblioteca (playlists, favoritos, historial) |
| `Esc`     | Cancelar / salir del modo actual |
| `q`       | Salir           |

### Modo biblioteca (`L`)

| Tecla     | Acción          |
|-----------|-----------------|
| `↑` / `↓` | Navegar la sección activa |
| `n` / `p` | Cambiar de sección (Playlists / Favoritos / Historial) |
| `Enter`   | Reproducir la playlist o pista seleccionada |
| `f`       | Alternar favorito de la pista seleccionada |
| `a`       | Añadir la pista seleccionada a una playlist |
| `c`       | Crear una playlist nueva (introduce el nombre y confirma) |
| `Esc` / `L` | Volver al modo normal |

## Cómo funciona

```
TUI (Bubble Tea) → search (yt-dlp ytsearch) → cola → player (mpv IPC socket)
```

`mpv` se lanza una vez en modo `--idle` y se controla por un socket Unix con
comandos JSON. La resolución de audio la hace el hook yt-dlp integrado de mpv
(`--ytdl-format=bestaudio`). Al terminar una pista, mpv emite `end-file` y la cola
avanza automáticamente.

## Datos y configuración

- Config (opcional): `~/.config/terminaltube/config.yaml`
  ```yaml
  search_results: 10
  volume: 70
  mpv_path: mpv
  ytdlp_path: yt-dlp
  ```
- Biblioteca (playlists, favoritos, historial): `~/.local/share/terminaltube/library.db` (SQLite)
- Historial legado: `~/.local/share/terminaltube/history.json` se importa una sola vez a `library.db` y se conserva como `history.json.bak`
- Logs: `~/.local/state/terminaltube/terminaltube.log`

## Tests

```bash
go test ./...                      # unitarios (queue, search, history)
go test -tags live ./internal/player/   # IPC contra mpv real
```

## Roadmap

- **Fase 1 (MVP, actual):** búsqueda, reproducción, cola, atajos, historial.
- **Fase 2:** playlists, favoritos, persistencia SQLite.
- **Fase 3:** letras, portadas, caché, Discord Rich Presence.
