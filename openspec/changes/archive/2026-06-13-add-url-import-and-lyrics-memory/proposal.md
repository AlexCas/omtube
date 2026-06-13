# Proposal: Ingesta por URL y memoria de letra

## Intent

Hoy solo se puede añadir música buscando por texto, y la letra se resuelve con una
consulta automática que un título sucio (típico al pegar una URL de YouTube) puede no
acertar, sin forma de corregirla ni recordarla. Los usuarios quieren pegar una URL de
YouTube para encolarla, importar una playlist de YouTube como lista local, ajustar y
recordar con qué búsqueda se encontró la letra de una canción, y limpiar la cola de un
golpe.

## Scope

### In Scope
- **Añadir por URL**: pegar una URL de vídeo de YouTube, resolver su metadato y
  encolarla; ofrecer añadirla a una playlist existente (reusa el picker actual).
- **Importar playlist desde URL**: resolver una URL de playlist de YouTube, pedir un
  nombre de playlist local (tecleado por el usuario) y crearla con todas sus pistas.
- **Memoria de letra**: búsqueda manual de letra (el usuario teclea una consulta, ve
  candidatos de lrclib y elige); se guarda la consulta/referencia con la que se
  encontró la letra, vinculada a la canción, y se reusa al re-reproducirla.
- **Limpiar cola**: vaciar la cola completa y detener la reproducción.

### Out of Scope
- Edición manual del texto de la letra (solo se elige el candidato del proveedor).
- Descarga/caché masiva de playlists completas (se reusa la caché por reproducción).
- Quitar pistas individuales de la cola o reordenarla (solo limpiar todo).
- Sincronización con playlists de YouTube (la importación es una copia puntual).

## Capabilities

### New Capabilities
- None (todas las capacidades afectadas ya existen).

### Modified Capabilities
- `youtube-search`: además de texto libre, resolver una URL de vídeo única y una URL
  de playlist a resultados estructurados.
- `playback-queue`: limpiar la cola completa.
- `playlists`: importar una playlist desde una URL de YouTube como lista local.
- `lyrics`: búsqueda manual de letra con selección de candidato; persistir y reusar la
  consulta/referencia con la que se encontró la letra de cada canción.
- `tui-shell`: modos/teclas nuevos para entrada de URL, importación de playlist,
  búsqueda de letra y limpiar cola.

## Approach

Reutilizar `yt-dlp` desde `internal/search` para resolver URLs: `Resolve(url)` (vídeo
único, metadato completo) y `ResolvePlaylist(url)` (entradas vía `--flat-playlist`),
con un clasificador de URLs en `internal/search/url.go`. `queue.Clear()` y un nuevo
`player.Stop()` cubren limpiar-y-parar. La letra gana búsqueda de candidatos
(`/api/search`) y selección manual; `lyrics_cache` se amplía (migración 3 aditiva) con
`query` y `provider_id` para recordar y reusar la referencia al re-reproducir. La UI
añade modos para los nuevos flujos reusando el `textinput` y el `picker` existentes.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/search/url.go` | New | Clasificar URL de YouTube (vídeo/playlist) y extraer ids |
| `internal/search/ytdlp.go` | Modified | `Resolve(url)` y `ResolvePlaylist(url)` |
| `internal/queue/queue.go` | Modified | `Clear()` |
| `internal/player/player.go`/`mpv.go` | Modified | `Stop()` |
| `internal/lyrics/lyrics.go` | Modified | Candidatos + selección manual + reuso de query guardada |
| `internal/storage/lyrics_cache.go`/`migrate.go` | Modified | Migración 3: `query`, `provider_id` |
| `internal/ui/*` | Modified | Modos/teclas/mensajes para URL, import, letra y limpiar cola |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| yt-dlp lento/falla en URL o playlist grande | Med | Timeout y error legible; la TUI sigue operativa |
| Formatos de URL variados (watch/youtu.be/shorts/`list=`) | Med | Clasificador con tests de tabla por formato |
| Migración 3 rompe bibliotecas existentes | Low | Migración aditiva e idempotente; columnas con default |
| URL de vídeo con `list=` ambigua (¿vídeo o playlist?) | Med | Regla explícita: `watch?v=`+`list=` ⇒ vídeo; `/playlist?list=` ⇒ playlist |

## Rollback Plan

Revertir el commit/PR. La migración 3 solo AÑADE columnas a `lyrics_cache` (con
default), así que un binario anterior sigue funcionando ignorándolas; no hay pérdida de
datos de biblioteca.

## Dependencies

- `yt-dlp` y `mpv` en PATH (ya requeridos).
- lrclib (ya usado por la capacidad `lyrics`).

## Success Criteria

- [ ] Pegar una URL de vídeo la encola y permite añadirla a una playlist existente.
- [ ] Importar una URL de playlist crea una lista local (nombre tecleado) con sus pistas.
- [ ] El usuario puede buscar la letra manualmente, elegir un candidato y verla; al
      re-reproducir la canción, se reusa la consulta/referencia guardada.
- [ ] Limpiar la cola la vacía y detiene la reproducción.
- [ ] `go build ./...` y `go test ./...` en verde.
