# Omusic

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

### Homebrew (Linux y macOS)

```bash
brew install AlexCas/tap/omusic
```

Esto instala el binario `omusic` y, como dependencias, `yt-dlp` y `mpv`
(`chafa` es opcional, para el panel de portada).

### Desde el código

```bash
go build -o omusic .
# o instalar en $GOBIN:
go install .
```

## Uso

```bash
omusic
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

  # Enriquecimiento (Fase 3). Todos los toggles son opcionales; con todos
  # apagados la app se comporta exactamente como en la Fase 2.
  cache:
    enabled: true        # descarga el audio a disco para reproducirlo sin re-streamear
    max_size_mb: 1024    # límite de tamaño total de la caché (<=0 sin límite)
    max_age_days: 30     # antigüedad máxima por entrada (<=0 sin límite)
  lyrics:
    enabled: true        # panel de letra (lrclib); resalta la línea si es sincronizada
  artwork:
    enabled: true        # panel de portada (chafa; degrada a placeholder si no hay chafa)
  presence:
    enabled: false       # presencia "escuchando" en Discord
    app_id: ""           # requerido: tu propia Discord Application ID (sin app_id queda inactiva)
  ```
- Biblioteca (playlists, favoritos, historial): `~/.local/share/terminaltube/library.db` (SQLite)
- Caché de audio: `~/.cache/terminaltube/audio/` (vaciable con `rm -rf ~/.cache/terminaltube/`)
- Historial legado: `~/.local/share/terminaltube/history.json` se importa una sola vez a `library.db` y se conserva como `history.json.bak`
- Logs: `~/.local/state/terminaltube/terminaltube.log`

### Paneles de enriquecimiento

- **Letra:** se muestra bajo los paneles de resultados/cola cuando `lyrics.enabled`. Si la
  letra es sincronizada, la línea actual se resalta según la posición de reproducción; si
  no hay letra disponible, muestra `sin letra`.
- **Portada:** se muestra cuando `artwork.enabled`, renderizada con `chafa` (bloques/ASCII).
  Reutiliza la miniatura cacheada localmente cuando existe y solo descarga la miniatura
  remota de YouTube ante un miss. Si `chafa` no está instalado degrada a `[sin portada]`.
  El render nativo kitty/sixel es una mejora futura: actualmente la app usa chafa o degrada
  sin portada, y la detección nunca selecciona un backend que no pueda dibujar.
- **Indicador de caché:** las pistas con archivo local en caché muestran un `⤓` a la
  izquierda en los paneles de resultados y de cola.

## Tests

```bash
go test ./...                      # unitarios (queue, search, history, caché, letra, portada, presencia, UI)
go test -tags live ./internal/player/   # IPC contra mpv real
```

## Roadmap

- **Fase 1 (MVP, actual):** búsqueda, reproducción, cola, atajos, historial.
- **Fase 2:** playlists, favoritos, persistencia SQLite.
- **Fase 3:** letras, portadas, caché, Discord Rich Presence.
