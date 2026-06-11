# Exploration: add-media-enrichment (Fase 3)

## Punto de partida

Tras Fase 2 (biblioteca + SQLite), esta fase enriquece la experiencia: caché de
descargas, letras, portadas/thumbnails y presencia en Discord. Depende de la capa de
persistencia de Fase 2.

## Reuso

- `internal/storage` (Fase 2) guarda metadatos cacheados (rutas de archivos, letras).
- `internal/player` ya expone estado (pista actual, pos/dur) → fuente para Discord RPC.
- `internal/config` para rutas de caché (XDG cache) y toggles de cada feature.

## Hallazgos técnicos

- **Caché de descargas:** `yt-dlp` puede descargar el audio a un archivo en
  `~/.cache/terminaltube/`; mpv reproduce el archivo local si existe (evita re-resolver
  y re-descargar). Política de expiración por tamaño/antigüedad.
- **Letras:** API comunitaria sin auth, p.ej. lrclib.net (letras sincronizadas .lrc);
  fallback a no-sincronizadas. Mostrar en un panel; resaltar línea por `time-pos`.
- **Portadas/thumbnails:** en terminal requiere protocolo gráfico — kitty graphics o
  sixel; alternativa `chafa` (ASCII/blocks). Omarchy usa terminales compatibles (kitty
  /ghostty). Tratar como degradación elegante si el terminal no soporta imágenes.
- **Discord Rich Presence:** IPC local de Discord vía librería Go
  (p.ej. `github.com/hugolgst/rich-go`); publicar "escuchando: <título>".

## Decisiones heredadas

- SponsorBlock: NO se implementa.
- Audio sigue resolviéndose con el hook yt-dlp de mpv (caché añade ruta local opcional).
