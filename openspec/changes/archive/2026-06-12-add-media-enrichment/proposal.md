# Proposal: Media Enrichment (Fase 3)

## Intent

Con la biblioteca ya persistida (Fase 2), los usuarios quieren una experiencia más
rica: que las canciones repetidas carguen al instante, ver la letra mientras suena,
mostrar la portada y compartir en Discord lo que escuchan. Esta fase añade caché,
letras, portadas y Discord Rich Presence.

## Scope

### In Scope
- Caché de descargas: guardar el audio localmente y reproducir el archivo si existe.
- Letras: obtener y mostrar la letra (sincronizada si está disponible).
- Portadas/thumbnails: mostrar la carátula en terminales compatibles.
- Discord Rich Presence: publicar la pista en reproducción.
- Toggles en config para activar/desactivar cada feature.

### Out of Scope
- Edición/corrección manual de letras o metadatos.
- Descarga masiva de playlists completas (solo caché por reproducción).
- SponsorBlock (descartado por el usuario).

## Capabilities

### New Capabilities
- `download-cache`: descarga/expira audio local y lo prioriza en reproducción.
- `lyrics`: obtención y visualización de letras (con sync por tiempo si existe).
- `artwork`: render de portada en terminal (kitty/sixel/chafa, con degradación).
- `discord-rich-presence`: presencia "escuchando" vía IPC de Discord.

### Modified Capabilities
- `audio-playback`: usar archivo cacheado cuando exista; emitir cambios de pista para RPC.
- `tui-shell`: paneles de letra y portada; indicadores de caché.

## Approach

Nuevo paquete `internal/cache` (descarga vía yt-dlp a XDG cache + índice en SQLite),
`internal/lyrics` (cliente HTTP a API comunitaria), `internal/artwork` (detección de
protocolo gráfico del terminal) e `internal/presence` (Discord IPC). El player
consulta la caché antes de stremear; un suscriptor de eventos de player actualiza
letra sincronizada y presencia.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/cache` | New | Descarga, índice e invalidación |
| `internal/lyrics` | New | Cliente de letras + parser .lrc |
| `internal/artwork` | New | Render de imagen en terminal |
| `internal/presence` | New | Discord Rich Presence (IPC) |
| `internal/player` | Modified | Preferir archivo local; eventos de pista |
| `internal/ui` | Modified | Paneles letra/portada; estado de caché |
| `internal/config` | Modified | Toggles y rutas de caché |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Terminal sin soporte de imágenes | High | Degradar a sin-portada o `chafa` ASCII |
| API de letras caída/sin match | Med | Fallback a "sin letra"; cache de resultados |
| Discord no presente/cerrado | Med | Feature opcional; fallar en silencio |
| Caché crece sin límite | Med | Límite por tamaño/antigüedad configurable |

## Rollback Plan

Cada feature es opcional y desactivable por config. Revertir = apagar toggles y
borrar `~/.cache/terminaltube/`. No afecta a la biblioteca SQLite de Fase 2.

## Dependencies
- Fase 2 (`add-library-and-persistence`) completada (SQLite/storage).
- Librería Discord IPC (p.ej. `rich-go`); API de letras (p.ej. lrclib).
- Terminal con protocolo gráfico (kitty/sixel) o `chafa` para portadas.

## Success Criteria
- [ ] Reproducir una pista ya cacheada no vuelve a descargar/resolver.
- [ ] Se muestra la letra; si hay .lrc, resalta la línea según el tiempo.
- [ ] Portada visible en terminal compatible; degradación limpia si no.
- [ ] Discord muestra la pista en reproducción cuando está activo.
- [ ] Cada feature se puede desactivar por config sin romper la app.
