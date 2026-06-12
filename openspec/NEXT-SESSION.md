# Handoff — Continuar TerminalTube en una sesión nueva

> Pega esto (o ábrelo) al iniciar una sesión nueva para retomar el trabajo.

## Estado actual (2026-06-11)

- **Fase 1 (MVP): COMPLETADA, juzgada (APPROVED) y ARCHIVADA.**
  Archivo: `openspec/changes/archive/2026-06-11-bootstrap-terminaltube-mvp/`
  (ciclo SDD completo hasta `archive`). Specs maestras en `openspec/specs/`.
- **Fase 2 (Biblioteca y persistencia): COMPLETADA, juzgada (APPROVED round 2) y ARCHIVADA.**
  Archivo: `openspec/changes/archive/2026-06-11-add-library-and-persistence/`
  (ciclo SDD completo hasta `archive`; 24/24 tareas, 33/33 tests PASS).
  Specs maestras sincronizadas en `openspec/specs/` (playlists, favorites,
  library-persistence, playback-history, tui-shell).
  Deuda diferida (no bloqueante): tests automatizados de UI (`internal/ui`, 0% cobertura;
  7 escenarios de `tui-shell` UNTESTED) y migración no data-driven (1 sola migración hoy).
- **Fase 3 (Media enrichment): COMPLETADA, juzgada (APPROVED round 2) y ARCHIVADA.**
  Archivo: `openspec/changes/archive/2026-06-12-add-media-enrichment/`
  (ciclo SDD completo hasta `archive`; 28/28 tareas, 89/89 tests PASS).
  Entregada en cadena de 3 slices (cache/storage → lyrics/artwork/presence → integración UI).
  Specs maestras nuevas en `openspec/specs/`: download-cache, lyrics, artwork,
  discord-rich-presence; actualizadas: audio-playback, tui-shell.
  Código en rama `feature/media-enrichment` (NO commiteado todavía).
  Deuda diferida (no bloqueante): W3 tests teatest de UI (hay 9 tests model-level nuevos,
  pero sin golden frames de TTY simulado); render nativo kitty/sixel es trabajo futuro
  (hoy degrada a chafa o sin portada); falta smoke manual en TTY real (mpv/Discord/imágenes).
- **Fase 4 (Metadata enrichment): PREPARADA/ENCOLADA — explore + propose completados.**
  Change: `openspec/changes/add-metadata-enrichment/` (state: `propose/completed`).
  **Siguiente fase a ejecutar cuando se retome: `spec`.**
  Aborda 3 gaps de Fase 3 sobre fuentes MV de YouTube:
    1. Normalización de metadata (título/artista limpios) → nueva capability `metadata`, SOLO-query
       (no muta Title/Uploader guardados).
    2. Fallback de letra: lrclib `/api/search` sobre la query normalizada (sin deps externas).
    3. Portada real: MusicBrainz + Cover Art Archive (UA + throttle ~1 req/s + cache pos/neg,
       thumbnail YT como fallback) → capability `artwork` MODIFICADA.
  Restricciones: puro Go / no-cgo / binario estático; toggles opt-in; degradación elegante.
  Decisiones humanas ya tomadas (no re-litigar): fallback solo lrclib; portada MB+CAA (no iTunes);
  normalización solo-query. SponsorBlock sigue fuera.

## Qué ya está construido (Fase 1)

Binario `terminaltube` (Go, módulo `github.com/alexcasdev/terminaltube`):

```
main.go                  valida deps (yt-dlp, mpv), arranca mpv + TUI
internal/config          Viper + rutas XDG
internal/logging         Zap a archivo
internal/search          yt-dlp ytsearch → resultados
internal/player          mpv en --idle + cliente IPC JSON (socket Unix)
internal/queue           cola actual/next/prev (sin wrap)
internal/history         historial en JSON  ← Fase 2 lo migra a SQLite
internal/ui              Bubble Tea + Lip Gloss
```

Funciona: buscar, reproducir, cola con auto-avance, atajos, historial JSON.

## Cómo retomar la Fase 3 (en la sesión nueva)

1. Cerrar el **preflight SDD** (hard gate de `CLAUDE.md`): ritmo, artefactos, PRs, presupuesto.
2. Leer `openspec/changes/add-media-enrichment/proposal.md` y `exploration.md`.
3. Verificar estado con la skill `harness-workflow` (debe permitir `propose → spec`).
4. Ejecutar las fases restantes con los sub-agentes `sdd-*`:
   `spec → design → tasks → apply → verify` (y `judge`/`archive` si se desea),
   con Human Review Gate después de cada artefacto editable.
5. Actualizar `state.yaml` del change en cada transición.

## Comandos útiles

```bash
cd /home/alexcasdev/Projects/omarchymusic
go build ./... && go vet ./... && go test ./...     # sanity
go test -tags live ./internal/player/               # IPC contra mpv real
go build -o terminaltube . && ./terminaltube        # ejecutar la TUI
```

## Decisiones ya tomadas (no re-litigar)

- Lenguaje/stack: Go + Bubble Tea + Lip Gloss + mpv (IPC socket) + yt-dlp + Viper + Zap.
- Resolución de audio: hook yt-dlp de mpv (`--ytdl-format=bestaudio`), no `--get-url`.
- Anuncios de YouTube: no aparecen (se stream el audio crudo, no el player web).
- **SponsorBlock: descartado** por decisión del usuario (no implementar).
