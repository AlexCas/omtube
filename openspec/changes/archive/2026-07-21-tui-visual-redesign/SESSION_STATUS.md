# SESSION_STATUS

## Cambio activo
- **Nombre**: tui-visual-redesign
- **Descripción**: Rediseño visual de la TUI de Omusic conservando los mismos elementos: dashboard responsivo al tamaño de terminal (ancho y alto) y eliminación de fondos opacos que rompen las terminales translúcidas (vidriado).

## Fase actual
- **Fase**: archive (in-progress). 2026-07-21. Las 3 slices MERGEADAS (#22, #23, #24 en b5a0cfd).
- **Rama**: chore/archive-tui-visual-redesign (base master).
- **Objetivo archive**: mover openspec/changes/tui-visual-redesign/ → openspec/changes/archive/2026-07-21-tui-visual-redesign/, mover este SESSION_STATUS.md ahí y quitarlo del root, escribir archive-report.md.
- **PRs**: #22 (slice1) #23 (slice2) #24 (slice3) — todos MERGEADOS. Sin co-autores.
- **Jueces Slice 3**: APROBADO (2 ciegos en worktrees aislados, 0 bloqueantes; sin incidente). verify = PASA.
- **INFO futuro (ticket)**: a anchos ≤~40-47 el footer de ayuda del modal modeResults desborda; PREEXISTENTE/upstream (idéntico en list por defecto), no introducido por Slice 3. Fuera de alcance.
- **PENDIENTE**: al mergear #24 → fase ARCHIVE (mover artefactos a openspec/changes/archive/, mover SESSION_STATUS al folder archivado y quitarlo del root).

## Apply Slice 3 — completado 2026-07-21
- Archivos: styles.go (+30/-1), view.go (+26/-2), update_test.go (+5/-4, solo TestRenderQueueWindowsLongQueue → "▼ 80 más"), view_test.go (+145), testdata/view_results_120x30.golden (nuevo). 212 líneas (< 400).
- Delegate translúcido (selección por color/negrita/borde lateral │, sin relleno); themedList reemplaza list.Styles.Title (que traía Background("62")). maxQueueRows techo 20. build/vet/test ./... verdes; goldens de modeNormal sin cambios.
- **Decisión de usuario (vigente)**: umbral narrow <90 (portada oculta a 80 cols).
- **Aclaración al usuario**: "Caelestia" NO es dependencia externa (no está en go.mod); colores hardcodeados en styles.go; sin backup necesario; fallback = degradación automática de lipgloss.
- **RECORDATORIO**: correr los jueces de Slice 3 con isolation: worktree (por el incidente de git checkout en Slice 2).

## Slice 3 — tasks (resumen ejecutivo)
9 tareas en orden: S3-T1 delegate Caelestia (styles.go+view.go ~52 líneas) →
S3-T2 techo maxQueueRows 10→20 (view.go, 1 línea) → S3-T3 actualizar
TestRenderQueueWindowsLongQueue (update_test.go, ~8 líneas) → S3-T4 test paleta
hexadecimal (view_test.go, ~47 líneas) → S3-T5 assert no-Background delegate
(view_test.go, ~30 líneas) → S3-T6 test library translúcida (view_test.go, ~29 líneas)
→ S3-T7 golden modal resultados (view_test.go ~30 líneas + testdata nuevo) →
S3-T8 reconciliar comentario D4 (view.go, ~4 líneas) → S3-T9 verificación final.
Estimación total: ~175–230 líneas (< 400).

## Slice 2 — CERRADA (mergeada)
- PR #23 MERGEADO (2026-07-21, 1f835d8). En master: PlaceVertical(bodyH), columnas por breakpoint, portada oculta <90, golden 60×20.

## Slice 2 — CERRADA (aprobada por jueces)
- verify = PASA-CON-OBSERVACIONES; jueces = APROBADO (2 ciegos, 0 bloqueantes).
- 287 líneas de código (< 400). PR #23 abierto.

## Deuda para Slice 3
- Subir techo maxQueueRows 10→20 y actualizar TestRenderQueueWindowsLongQueue (update_test.go:525) que hoy lo fuerza.
- Obs-1 (verify Slice 1): test que afirme los hex de la paleta por nombre.
- Reconciliar design.md D4 (texto dice bodyH-2/techo 20; el código usa bodyH-5/techo 10).
- Alcance Slice 3: reestilizar modales/pickers (modeResults, biblioteca, pickers de letra) sin fondo de fila opaco; delegate de bubbles/list solo fg/borde; extender assert no-bg al modal.

## Apply Slice 2 — completado 2026-07-21
- Archivos: view.go (+87/-44), view_test.go (+143/-13), testdata/{view_60x20 (nuevo), view_80x24, view_120x30} (regen). 287 líneas de código (< 400).
- build/vet/test ./... verdes; 3 goldens difieren por pares; sin .got.
- PUNTO A DECIDIR (tras jueces): con narrow <90, a 80 cols la portada queda OCULTA (2 col). Por diseño, pero 80 es común → confirmar umbral con el usuario.
- Desviaciones justificadas: techo maxQueueRows=10 (test preexistente TestRenderQueueWindowsLongQueue lo exige), PlaceVertical en vez de Place ancho-completo (paridad Fase 2), chrome fijo=11+ayuda medida.

## Slice 1 — CERRADA
- PR #22 MERGEADO (2026-07-21, merge 1cdddb7). En master: styles.go sin Background opaco, computeLayout, tests, goldens.

## Preflight (decisiones de sesión)
- Ritmo: interactive
- Artefactos: openspec
- PRs: ask-always (presupuesto 400 líneas)
- Revisión: 400 líneas
- Playwright: no

## Decisiones de diseño confirmadas por el usuario
- Dirección: dashboard responsivo (opción C) que aprovecha el alto de la terminal.
- Paleta: conservar acentos mauve/teal (#e0aaff / #00f5d4). Sin temas por ahora.
- Fondos opacos #1a1a2e: eliminar (transparente por defecto para respetar el vidriado).
- Entrega en 3 SLICES ENCADENADAS (PRs < 400 líneas c/u):
  - Slice 1 (HECHA): transparencia + anchos fluidos + freno de desborde.
  - Slice 2: jerarquía dashboard + uso del alto (regiones proporcionales); ocultar portada en narrow; golden 60×20.
  - Slice 3: reestilizar modales/pickers (resultados, biblioteca, letra).
- Breakpoints fijados: narrow < 90 · medium 90–119 · wide ≥ 120.

## Fases completadas (Slice 1)
- explore — 2026-07-21 — exploration.md
- propose — 2026-07-21 — proposal.md (capability modificada: caelestia-ui)
- spec — 2026-07-21 — specs/caelestia-ui/spec.md + caelestia-ui.feature (@slice1/@slice2/@slice3)
- design — 2026-07-21 — design.md (breakpoints 90/120; assert GetBackground()==""; computeLayout)
- tasks — 2026-07-21 — tasks.md (T1–T5, Slice 1)
- apply — 2026-07-21 — styles.go (-2), view.go (+233/-45), view_test.go (+85), goldens regen; ~320 líneas; build/vet/test verdes
- verify — 2026-07-21 — verify-report.md (PASA-CON-OBSERVACIONES)
- judge — 2026-07-21 — APROBADO

## Pendientes diferidos (para próximas slices)
- Obs-2 (verify + juez): test de frontera de computeLayout/classify a 59/60/89/90/119/120 → Slice 2 (toca esa función).
- Obs-1 (verify): test que afirme los hex de la paleta por nombre → Slice 3.
- Nota juez: en terminales < 40 cols (fuera de soporte) hay desborde por el piso minUsable=40.
- Desviaciones de apply (todas validadas por verify+jueces): artW>0 en narrow (portada se oculta hasta Slice 2), truncado cola queueW-6, progressW descuenta título, wrappers de firmas para update_test.go, hasNoBackground acepta NoColor{}.

## Artefactos / rutas clave
- openspec/changes/tui-visual-redesign/ — exploration.md, proposal.md, design.md, tasks.md, verify-report.md, specs/caelestia-ui/{spec.md, caelestia-ui.feature}, state.yaml
- internal/ui/styles.go, view.go, view_test.go, testdata/*.golden — Slice 1 aplicada

## Tareas de Slice 2 — resumen (tasks.md)

| Tarea | Alcance | Est. líneas |
|-------|---------|-------------|
| S2-T1 | Altura dinámica en computeLayout (bodyH, maxQueueRows, lyricWindow) | ~15–20 mod |
| S2-T2 | Narrow 2-col / medium-wide 3-col; consumir showArtwork | ~25–35 mod |
| S2-T3 | Place/PlaceVertical en renderMiddleSection | ~20–30 mod |
| S2-T4 | Tests de frontera (classify/computeLayout) + Test60x20NarrowNoArtwork | ~80–100 add |
| S2-T5 | Golden 60×20 + regenerar 80×24 / 120×30 | ~5 add + regen |
| Total | | ~145–190 líneas (< 400) |

Re-slicing 2a/2b disponible en design.md si la implementación supera 400 líneas.

## Siguiente paso recomendado
- Human Review Gate: mostrar tasks de Slice 2 al usuario y esperar aprobación.
- Al aprobar: invocar apply (Slice 2) en rama feat/tui-redesign-slice2.
- Preflight de sesión ya fijado (interactive/openspec/ask-always/400/no-playwright); no repetir.
