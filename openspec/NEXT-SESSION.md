# Handoff — Continuar TerminalTube en una sesión nueva

> Pega esto (o ábrelo) al iniciar una sesión nueva para retomar el trabajo.

## Estado actual (2026-06-11)

- **Fase 1 (MVP): COMPLETADA, juzgada (APPROVED) y ARCHIVADA.**
  Archivo: `openspec/changes/archive/2026-06-11-bootstrap-terminaltube-mvp/`
  (ciclo SDD completo hasta `archive`). Specs maestras en `openspec/specs/`.
- **Fase 2: PROPUESTA LISTA, pendiente de continuar.**
  Change: `openspec/changes/add-library-and-persistence/` (state: `propose/completed`).
  **Siguiente fase a ejecutar: `spec`.**
- **Fase 3: ENCOLADA.**
  Change: `openspec/changes/add-media-enrichment/` (state: `propose/completed`).
  No empezar hasta cerrar Fase 2.

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

## Cómo retomar la Fase 2 (en la sesión nueva)

1. Leer `openspec/changes/add-library-and-persistence/proposal.md` y `exploration.md`.
2. Verificar estado con la skill `harness-workflow` (debe permitir `propose → spec`).
3. Ejecutar las fases restantes con los sub-agentes `sdd-*`:
   `spec → design → tasks → apply → verify` (y `judge`/`archive` si se desea).
4. Actualizar `state.yaml` del change en cada transición.

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
