# Exploration: add-library-and-persistence (Fase 2)

## Punto de partida

Fase 1 (`bootstrap-terminaltube-mvp`) entregó búsqueda, reproducción (mpv IPC), cola,
atajos e historial en JSON. Esta fase añade la **biblioteca personal** (playlists,
favoritos) y migra la persistencia a **SQLite**.

## Reuso desde el MVP

- `internal/search.Result` es el modelo de pista; reutilizar como base de filas.
- `internal/queue.Queue.Add` permite cargar una playlist como cola sin cambios.
- `internal/config` ya resuelve rutas XDG (`DataDir`) → ubicar la DB ahí.
- `internal/history` ya define `Entry`; su persistencia JSON se sustituye por SQLite.

## Hallazgos técnicos

- **Driver SQLite:** usar `modernc.org/sqlite` (pure Go, sin cgo) para conservar la
  ventaja de "binario único" del stack. Evitar `mattn/go-sqlite3` (requiere cgo).
- Esquema candidato: tablas `tracks`, `playlists`, `playlist_tracks`, `favorites`,
  `history` con `video_id` como clave natural de pista.
- La UI necesitará un selector/vista de biblioteca (nuevo panel o modo) y comandos
  para crear playlist / añadir a playlist / marcar favorito.

## Decisiones heredadas

- SponsorBlock: NO se implementa (decisión del usuario).
- Resolución de audio sigue por el hook yt-dlp de mpv.
