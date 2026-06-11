# Archive Report: add-library-and-persistence (Fase 2)

**Archived**: 2026-06-11
**Mode**: openspec
**SDD cycle**: explore → propose → spec → design → tasks → apply → verify → judge → archive (todas completadas)

## Change Summary

Fase 2 introduce la capa de biblioteca y persistencia de TerminalTube: una base de datos
SQLite local de un solo archivo (driver puro Go `modernc.org/sqlite`, sin cgo, preservando
el binario único) que respalda pistas, playlists, favoritos e historial. Se añade un esquema
versionado con migraciones idempotentes (`PRAGMA user_version`), repositorios CRUD por
entidad, dominios de playlists y favoritos, un modo biblioteca en la TUI (navegar playlists/
favoritos/historial, crear playlist, alternar favorito, añadir a playlist, reproducir como
cola), y la migración del historial JSON heredado al nuevo almacenamiento SQLite. El
historial pasa de persistir en `history.json` a persistir en `library.db`.

## 2-PR Chain Delivery

Entregado como cadena ordenada de 2 work units (feature-branch-chain, exception-ok por
presupuesto de >400 líneas), aplicada secuencialmente con cada unidad compilando y en verde
antes de la siguiente:

| Unit | PR | Goal | Tasks |
|------|----|------|-------|
| 1 | PR 1 | Backend de persistencia: storage layer + repos + dominio playlist/favoritos + unit tests | 1.1–1.7, 2.1, 2.2, 4.1–4.4 |
| 2 | PR 2 | Integración y UI: historial-sobre-repo + import JSON, ruta config, wiring en main, modo biblioteca, tests de integración, docs | 2.3, 3.1–3.6, 4.5, 5.1, 5.2 |

## Task Completion Gate

- `tasks.md`: 24/24 tareas marcadas `[x]`, 0 sin marcar. ✅
- `verify-report.md`: 0 issues CRITICAL. ✅

## Verify Result

**PASS WITH WARNINGS** (re-verificado post-judge round 1, sin CRITICAL).

- `go build ./...`, `go vet ./...` → exit 0 (limpios).
- `gofmt -l .` → vacío (sin archivos sin formatear).
- Binario estático / no-cgo (criterio de éxito del proposal): `CGO_ENABLED=0 go build` →
  ELF estáticamente enlazado, `ldd` → not a dynamic executable. ✅
- `go test ./... -count=1` → **33/33 PASS / 0 FAIL / 0 SKIP** (subió de 30 tras los fixes
  de judge).
- 27/34 escenarios de spec ✅ COMPLIANT, 2 ⚠️ PARTIAL, 7 ❌ UNTESTED (todos UI tui-shell).

## Judge Result

**APPROVED — round 2** (judgment-day dual review).

- Round 1: ISSUES FOUND — 0 CRITICAL, 3 WARNINGs confirmados por ambos jueces.
- Fixes aplicados y re-verificados (código real + tests que afirman el comportamiento):
  - **A-W1** — Upsert clobbreaba duration/title/uploader: `tracks.go` ahora guarda cada
    campo con `CASE WHEN excluded.X <> '' / > 0 THEN excluded.X ELSE tracks.X END`. Test
    `TestTrackRepoUpsertDoesNotBlankFields`.
  - **B-W1** — `history.json` corrupto bloqueaba el arranque: `importLegacyJSON` ahora
    respalda el archivo y continúa con historial vacío en vez de propagar el error. Test
    `TestCorruptLegacyJSONDoesNotBrickStartup`.
  - **B-W2** — Import masivo no transaccional: import envuelto en una sola tx
    (`UpsertTx`/`InsertTx`/`Commit`) y solo entonces backup. Test `TestTxInsertsAreAllOrNothing`.
- Re-verify round 2: build/vet/gofmt/test en verde, sin regresiones → APPROVED por ambos jueces.

## Specs Synced

Las delta del change se fusionaron en las specs maestras de `openspec/specs/`:

| Domain | Created/Updated | Requirements |
|--------|-----------------|--------------|
| playlists | Created (nueva capability) | 5 (Create, Rename, Delete, Manage Tracks, Play as Queue) |
| favorites | Created (nueva capability) | 3 (Toggle, List, Persist) |
| library-persistence | Created (nueva capability) | 4 (Single-File Local DB, Versioned Schema, Track Identity, Entity Repositories) |
| playback-history | Updated (1 MODIFIED + 2 ADDED) | 4 (Record Played Tracks [preservado], Persist to Local Database [rename], Migrate Legacy JSON History, Browse History) |
| tui-shell | Updated (3 ADDED) | 7 (4 previos preservados + Library Mode, Create Playlist from UI, Library Action Shortcuts) |

### Reconciliación de playback-history (rename explícito)

El requirement canónico **"Persist to JSON File"** fue **reemplazado** (heading + escenarios)
por la versión MODIFIED del delta **"Persist to Local Database"**: el historial ya no persiste
en `history.json` sino en la base SQLite `library.db`. No quedó requirement huérfano ni
duplicado (verificado: `grep "Persist to JSON File"` en `openspec/specs/` → sin coincidencias).
El requirement **"Record Played Tracks"** se preservó intacto. Se añadieron **"Migrate Legacy
JSON History"** y **"Browse History"**. La sección Purpose se actualizó de "archivo JSON local,
sin base de datos" a almacenamiento SQLite con migración del JSON heredado.

## Known Issues (no bloqueantes, anotados — WARNING arrastrado)

- **WARNING arrastrado — UI tests (W1)**: `internal/ui` no tiene tests automatizados (0.0%
  cobertura); los 7 escenarios UI de tui-shell (Library Mode open/close + navigate, Create
  Playlist from UI por nombre + reject empty/duplicate, Library Action Shortcuts toggle
  favorite / add to playlist / play playlist) quedan UNTESTED-por-automatización, verificados
  solo por build + razonamiento manual de código. Los dominios subyacentes que invocan SÍ
  están testeados y en verde, por lo que el riesgo es de wiring/UX, no de lógica de dominio.
  Inalterado por el ciclo de fixes; recomendado cubrir con `teatest` o unit tests a nivel de
  modelo en una fase futura (consistente con el gap diferido de Fase 1).
- **PARTIAL (W3)**: el camino de migración multi-paso no es data-driven (solo existe 1
  migración hoy) y la persistencia de favoritos está probada a nivel de repo en disco, no vía
  un reopen completo de la app.

## Outcome

Fase 2 (biblioteca y persistencia) entregada en cadena de 2 PRs, verificada (PASS WITH
WARNINGS, sin CRITICAL), juzgada (APPROVED round 2 tras corregir A-W1/B-W1/B-W2) y archivada.
24/24 tareas completas. Source of truth actualizado en `openspec/specs/` (5 dominios). Único
warning vigente: ausencia de tests automatizados de UI (7 escenarios tui-shell), diferido a
fase futura.
