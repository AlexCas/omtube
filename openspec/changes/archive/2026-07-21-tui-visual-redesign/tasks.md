# Tasks — TUI Visual Redesign: Slice 1 (Base)

Scope: purely presentational changes to `internal/ui/styles.go`, `internal/ui/view.go`,
and `internal/ui/view_test.go`. No Model/Update/messages/keys/services touched.
Estimated changed lines: ~180–240 (well under 400-line budget).

---

## Order of execution

Tasks MUST be executed in numbered order. Dependencies are noted inline.

---

## T1 — Remove opaque backgrounds from styles.go

**File**: `internal/ui/styles.go`
**Depends on**: nothing (first change)
**Spec mapping**: @slice1 "No opaque background paints over the terminal glass"
  (spec.md: "Caelestia Palette", feature: "No opaque background paints over the terminal glass")

- [ ] T1.1 — Remove `.Background(lipgloss.Color("#1a1a2e"))` from the `title` style
      (currently line ~21). Keep `.Bold(true)`, `.Foreground(#e0aaff)`,
      `.Border(lipgloss.RoundedBorder())`, `.BorderForeground(#e0aaff)`, `.Padding(0,1)`.
- [ ] T1.2 — Remove `.Background(lipgloss.Color("#1a1a2e"))` from the `panel` style
      (currently line ~26). Keep `.Border(lipgloss.RoundedBorder())`,
      `.BorderForeground(#e0aaff)`, `.Padding(0,1)`.
- [ ] T1.3 — Verify `go build ./...` passes with no errors after this change.

Estimated lines changed: ~4 deletions.

---

## T2 — Add layout types and computeLayout to view.go

**File**: `internal/ui/view.go`
**Depends on**: T1 (build must be green before adding new code)
**Spec mapping**: @slice1 "Widths derive from runtime dimensions", "No rendered line exceeds terminal width"
  (design.md: Decision 1 — Breakpoint thresholds; Decision 3 — Fluid width formula)

- [ ] T2.1 — Add `breakpoint` type and constants at the top of `view.go` (after imports):
      ```go
      type breakpoint int
      const (
          bpNarrow breakpoint = iota // < 90 cols
          bpMedium                   // 90–119 cols
          bpWide                     // >= 120 cols
      )
      ```
- [ ] T2.2 — Add `classify(width int) breakpoint` function:
      - `width < 90`  → `bpNarrow`
      - `width < 120` → `bpMedium`
      - else          → `bpWide`
- [ ] T2.3 — Add `layout` struct with fields:
      `bp`, `queueW`, `lyricsW`, `artW`, `progressW`,
      `maxQueueRows`, `lyricWindow`, `plainLines`,
      `nowTitleTrunc`, `libLineTrunc`, `showArtwork int/bool`
      (exact types per design.md Decision 3).
- [ ] T2.4 — Add `computeLayout(width, height int) layout` function:
      - `usable = max(width-2, minUsable)` (define `minUsable = 40`)
      - Per breakpoint, compute `queueW`, `lyricsW`, `artW` as `round(usable * pct)`,
        clamped (queue min 24 / lyrics min 28 / artwork fixed 24–28).
      - Fold remainder into `lyricsW` so `queueW + lyricsW + artW == usable`
        (and `artW == 0` for `bpNarrow`).
      - `showArtwork = bp != bpNarrow`
      - `progressW = clamp(width-24, 8, 40)` (decorLen ~24: state+times+vol chrome).
      - `nowTitleTrunc = max(8, lyricsW-4)` (reasonable title trunc, revised at verify).
      - `maxQueueRows = 10` (Slice 1 keeps current value; dynamic from height is Slice 2).
      - `lyricWindow = 7` (Slice 1 keeps current value; dynamic from height is Slice 2).
      - `plainLines = 8` (Slice 1 keeps current value).
      - `libLineTrunc = max(20, width-4)`.
- [ ] T2.5 — Add unexported helper `clamp(v, lo, hi int) int` if not already present.
- [ ] T2.6 — Verify `go build ./...` passes after T2.

Estimated lines added: ~60–75.

---

## T3 — Apply fluid widths and truncations in view.go render helpers

**File**: `internal/ui/view.go`
**Depends on**: T2 (layout types must exist)
**Spec mapping**: @slice1 "Widths derive from runtime dimensions", "No rendered line exceeds terminal width"
  (design.md: File Changes — view.go; Decision 3)

- [ ] T3.1 — In `View()`, call `l := computeLayout(m.width, m.height)` at the top
      (after the early-return guards, before any render calls). Thread `l` into each
      render helper call.
- [ ] T3.2 — Change `renderQueue()` signature to `renderQueue(l layout) string`.
      Replace hardcoded `Width(36)` → `m.styles.panel.Width(l.queueW)`.
      Replace hardcoded truncation `28` → `l.queueW - 2`.
      Replace constant `maxQueueRows` → `l.maxQueueRows`.
- [ ] T3.3 — Change `renderNowPlaying()` signature to `renderNowPlaying(l layout) string`.
      Replace hardcoded `progressBar(..., 30)` → `progressBar(m.pos, m.dur, l.progressW)`.
      Replace hardcoded title truncation `32` → `l.nowTitleTrunc`.
- [ ] T3.4 — Change `renderLyricsPanel()` signature to `renderLyricsPanel(l layout) string`.
      Replace hardcoded `Width(50)` → `m.styles.panel.Width(l.lyricsW)`.
      Replace hardcoded plain-lyrics args `(48, 8)` → `(l.lyricsW-2, l.plainLines)`.
      Pass `l` into `renderSyncedLyrics(l layout)`.
- [ ] T3.5 — Change `renderSyncedLyrics()` signature to `renderSyncedLyrics(l layout) string`.
      Replace hardcoded `window = 7` → `l.lyricWindow`.
      Replace hardcoded truncation `46` → `l.lyricsW - 4`
      (extra 2 for the "▶ " prefix so the line stays within the inner width).
- [ ] T3.6 — Change `renderArtworkPanel()` signature to `renderArtworkPanel(l layout) string`.
      Replace hardcoded `Width(28)` → `m.styles.panel.Width(l.artW)`.
- [ ] T3.7 — Change `renderEnrichment()` signature to `renderEnrichment(l layout) string`.
      Pass `l` to `renderLyricsPanel(l)` and `renderArtworkPanel(l)`.
      NOTE: Slice 1 does NOT hide artwork for `bpNarrow` — that is Slice 2.
      Slice 1 only ensures widths are fluid so no overflow at 60/80/120.
- [ ] T3.8 — Change `renderMiddleSection()` signature to `renderMiddleSection(l layout) string`.
      Pass `l` to `renderQueue(l)` and `renderEnrichment(l)`.
- [ ] T3.9 — In `trackLines()` (renderLibList helper), replace hardcoded truncation `60`
      with `l.libLineTrunc` — requires passing `l` into `renderLibrary()` and
      `renderLibList()` as well, OR extracting the trunc constant from `trackLines`
      by making it accept a `maxCols int` parameter and calling it as
      `trackLines(m.libFavorites, l.libLineTrunc)`.
      Prefer the simpler second option to avoid cascading signature changes in library code.
- [ ] T3.10 — Verify `go build ./...` passes after T3.

Estimated lines changed: ~60–80 (signature changes + replacements).

---

## T4 — Add test assertions in view_test.go (BEFORE regenerating goldens)

**File**: `internal/ui/view_test.go`
**Depends on**: T3 (code must compile; asserts must pass against new behavior)
**Spec mapping**: @slice1 "No rendered line exceeds terminal width", "No opaque background paints over the terminal glass", "80x24 and 120x30 goldens differ"
  (design.md: Decision 2 — no-Background assert; Decision 5 — golden test strategy)

- [ ] T4.1 — Add `hasNoBackground(s lipgloss.Style) bool` helper in `view_test.go`:
      returns `s.GetBackground() == lipgloss.Color("")`.
- [ ] T4.2 — Add `TestStylesNoBackground(t *testing.T)`:
      constructs `defaultStyles()`, asserts `hasNoBackground(s.title)` and
      `hasNoBackground(s.panel)`. Maps to spec scenario "No opaque background paints
      over the terminal glass".
- [ ] T4.3 — Add `TestNoLineExceedsWidth(t *testing.T)`:
      table test over widths `[]int{60, 80, 120}`. For each:
      - create a test model with that width and a representative height (24).
      - call `m.View()`, split on `\n`.
      - assert `lipgloss.Width(line) <= width` for every non-empty line.
      Maps to spec scenario "No rendered line exceeds terminal width" (all three Examples).
- [ ] T4.4 — Add `TestGoldensDiffer(t *testing.T)`:
      reads `testdata/view_80x24.golden` and `testdata/view_120x30.golden` as bytes.
      Asserts `!bytes.Equal(want80, want120)`.
      NOTE: this test will fail until golden files are regenerated in T5. Mark with
      `t.Skip("run after UPDATE_GOLDEN=1")` initially, or add a guard:
      if either file is missing, `t.Skip(...)`.
      Maps to spec scenario "80x24 and 120x30 goldens differ".
- [ ] T4.5 — Run `go test ./internal/ui/... -run TestStylesNoBackground` — must pass.
- [ ] T4.6 — Run `go test ./internal/ui/... -run TestNoLineExceedsWidth` — must pass.
- [ ] T4.7 — Run `go test ./internal/ui/... -run TestViewGolden` — expected FAIL
      (goldens are stale; confirms the code changed). Note the failure is expected at
      this step.

Estimated lines added: ~55–70.

---

## T5 — Regenerate golden fixtures and verify full test suite

**Files**: `internal/ui/testdata/view_80x24.golden`, `internal/ui/testdata/view_120x30.golden`
**Depends on**: T4 (all non-golden asserts must pass first; goldens must be regenerated
against the new layout code)
**Spec mapping**: @slice1 "80x24 and 120x30 goldens differ", "Golden Determinism"
  (design.md: Decision 5 — golden test strategy; spec.md "Golden Determinism")

- [ ] T5.1 — Run `UPDATE_GOLDEN=1 go test ./internal/ui/... -run TestViewGolden`
      to regenerate `view_80x24.golden` and `view_120x30.golden`.
- [ ] T5.2 — Inspect the diff of both golden files:
      - Confirm `view_80x24.golden` uses narrower panel widths (bpNarrow or bpMedium).
      - Confirm `view_120x30.golden` uses wider panel widths (bpWide).
      - Confirm NO line in either file is visually wider than its target width.
      - Confirm the two files differ in content (different widths, different column counts).
- [ ] T5.3 — Remove the `t.Skip` guard from `TestGoldensDiffer` (added in T4.4).
- [ ] T5.4 — Run `go test ./internal/ui/...` (full suite, no UPDATE_GOLDEN):
      - `TestStylesNoBackground` — PASS.
      - `TestNoLineExceedsWidth` — PASS.
      - `TestGoldensDiffer` — PASS.
      - `TestViewGolden` (80x24, 120x30) — PASS.
      - `TestToggleOffParity_*` (existing parity tests) — PASS.
- [ ] T5.5 — Run `go build ./...` one final time to confirm the full module builds clean.

Estimated lines changed: golden files regenerated (not counted toward code budget).

---

## Summary table

| Task | Files | Est. lines | Spec/Scenario |
|------|-------|-----------|---------------|
| T1 Remove opaque bg | `styles.go` | ~4 del | Caelestia Palette, No opaque bg |
| T2 Layout types + computeLayout | `view.go` | ~65 add | Widths derive from runtime |
| T3 Fluid widths in render helpers | `view.go` | ~70 mod | No line exceeds width, Widths derive |
| T4 Test assertions | `view_test.go` | ~65 add | No bg assert, Width assert, Goldens differ |
| T5 Regen goldens + full suite | `testdata/*.golden` | regen | Golden Determinism, 80x24 != 120x30 |

**Total code lines (styles + view + test)**: ~200 lines changed/added (< 400 budget).
Golden files are regenerated, not authored — not counted toward budget.

---

## Verification checklist (for apply phase sign-off)

- [ ] `go build ./...` — green
- [ ] `go test ./internal/ui/...` — all tests pass
- [ ] `TestStylesNoBackground` — passes (no Background on title/panel)
- [ ] `TestNoLineExceedsWidth` at 60, 80, 120 — passes
- [ ] `TestGoldensDiffer` — passes (80x24 != 120x30)
- [ ] `TestViewGolden` 80x24 and 120x30 — pass (goldens match regenerated output)
- [ ] No Model/Update/messages/keys/services files touched
- [ ] No Slice 2 or Slice 3 code introduced

---

## Slice 2 — Dashboard / uso del alto

Scope: uso del alto, `maxQueueRows`/lyricWindow dinámicos, columnas por breakpoint
(narrow 2 col, medium/wide 3 col con proporciones), ocultar portada en narrow,
golden 60×20 y tests de frontera. Solo se tocan `internal/ui/view.go`,
`internal/ui/view_test.go` y `internal/ui/testdata/*.golden`. No se toca
`styles.go` (a menos que sea imprescindible), ni `Model`/`Update`/`messages`/
`keys`/servicios. Slice 3 queda fuera.

Estimación de líneas: ~230–310 (view.go ~100–130 mod, view_test.go ~80–100 add,
goldens regenerados no cuentan). Dentro del presupuesto de 400 líneas.
Si al implementar se supera 400 se aplica el re-slicing 2a/2b definido en design.md:
2a = solo altura/vertical fill sin 3 columnas; 2b = 3 columnas + artwork breakpoint
+ golden 60×20.

Las tareas DEBEN ejecutarse en orden numérico. Las dependencias se indican en cada una.

---

### S2-T1 — Extender computeLayout con dimensionamiento vertical

**Archivo**: `internal/ui/view.go`
**Depende de**: nada (primer cambio de Slice 2)
**Mapeo de escenario**: "Vertical space is used without clipping" (@slice2 @happy)

El objetivo es hacer que `maxQueueRows` y `lyricWindow` dependan de `bodyH`,
que a su vez se calcula midiendo el "chrome" (filas fijas de la vista).

- [ ] S2-T1.1 — Definir la constante `chromeRows = 7` como comentario documentado
      en `computeLayout`, que suma: 1 título, 1 línea vacía, 1 ahora-suena,
      1 línea vacía, 1 estado/búsqueda, 1 línea vacía, 1 ayuda, 1 visualizador.
      (El valor real tras contar en `View()` es title(1) + blank(1) + nowPlaying(1)
      + blank(1) + status(1) + blank(1) + help(1) + visualizer(1) = 8 filas de
      chrome + 2 líneas vacías de separación de la sección media = 10 filas total
      de chrome. Validar contando las llamadas `WriteString("\n")` en `View()`
      antes de y después de `renderMiddleSection`; ajustar el valor según lo
      encontrado.)
- [ ] S2-T1.2 — En `computeLayout(width, height int) layout`, calcular:
      `bodyH := max(height - chromeRows, minBody)` donde `minBody = 4`.
- [ ] S2-T1.3 — Reemplazar `maxQueueRows: 10` (constante) por:
      `maxQueueRows: clamp(bodyH-2, 3, 20)`.
      El `-2` descuenta el encabezado "Cola (N)" y el borde del panel.
- [ ] S2-T1.4 — Reemplazar `lyricWindow: 7` (constante) por:
      `lyricWindow: clamp(bodyH-2, 3, 12)`. Aplicar normalización a impar para
      que la línea activa quede centrada: si `lyricWindow % 2 == 0 { lyricWindow-- }`.
- [ ] S2-T1.5 — Reemplazar `plainLines: 8` por:
      `plainLines: clamp(bodyH-2, 3, 12)`.
- [ ] S2-T1.6 — Verificar `go build ./...` — debe pasar sin errores.

Estimación de líneas modificadas: ~15–20 (solo dentro de `computeLayout`).

---

### S2-T2 — Ocultar portada en narrow y aplicar columnas de 2/3 por breakpoint

**Archivo**: `internal/ui/view.go`
**Depende de**: S2-T1 (layout con `showArtwork` ya calculado; compilación verde)
**Mapeo de escenarios**:
- "Narrow breakpoint hides artwork" (@slice2 @edge)
- "Breakpoints render distinct deterministic layouts" (@slice2 @happy)
- "Queue and lyrics behaviors preserved" (@slice2 @happy)

La variable `showArtwork` ya está calculada en `computeLayout` desde Slice 1
(`showArtwork: bp != bpNarrow`). Esta tarea la consume en `renderEnrichment`.

- [ ] S2-T2.1 — En `renderEnrichment(l layout)`, antes de agregar el panel de
      portada, añadir la guarda:
      ```go
      if hasArtwork && l.showArtwork {
          panels = append(panels, m.renderArtworkPanelAt(l))
      }
      ```
      Reemplazar la lógica actual que agrega la portada incondicionalmente cuando
      `m.artwork != nil`.
- [ ] S2-T2.2 — En `computeLayout`, ajustar el bloque `bpNarrow` para distribuir
      el presupuesto en solo 2 columnas (cola + letra, sin portada):
      ```
      usable sin portada: budget = usable - 2*panelBorder  // 2 paneles, no 3
      queueW = clamp(round(budget * 0.42), qMin=24, 40)
      lyricsW = budget - queueW
      artW = 0
      ```
      Las proporciones 42%/58% producen a usable=78 (80 cols): queueW≈33, lyricsW≈45.
      Verificar que `queueW + lyricsW == budget` sin remanente (o plegarlo en lyricsW).
- [ ] S2-T2.3 — Para bpMedium (90–119 cols), confirmar que los porcentajes 34%/40%/26%
      (con 3 paneles y borde) producen `queueW+lyricsW+artW ≤ usable`.
      Si hay desfase por redondeo, plegar el remanente en `lyricsW`.
- [ ] S2-T2.4 — Para bpWide (≥120 cols), confirmar 30%/44%/26% y plegar remanente.
- [ ] S2-T2.5 — Verificar `go build ./...` — debe pasar.
- [ ] S2-T2.6 — Verificar `go test ./internal/ui/... -run TestNoLineExceedsWidth`
      — debe pasar en 60, 80, 120.

Estimación de líneas modificadas: ~25–35.

---

### S2-T3 — Place/PlaceVertical para uso del alto en la sección media

**Archivo**: `internal/ui/view.go`
**Depende de**: S2-T2 (breakpoints y showArtwork correctos; compilación verde)
**Mapeo de escenario**: "Vertical space is used without clipping" (@slice2 @happy)

El objetivo es que la banda central ocupe `bodyH` filas en lugar de su altura
natural, usando `lipgloss.Place`. Cuando `height > content` el bloque se centra
verticalmente; cuando `height < content` el contenido se recorta al mínimo
definido por los clamps de S2-T1.

- [ ] S2-T3.1 — Añadir el campo `bodyH int` al struct `layout` y asignarlo en
      `computeLayout` con el valor calculado en S2-T1.2.
- [ ] S2-T3.2 — En `renderMiddleSection(l layout)`, envolver el resultado final
      con:
      ```go
      band := lipgloss.JoinHorizontal(lipgloss.Top, queue, enrich)  // o solo queue
      return lipgloss.Place(l.queueW+l.lyricsW+l.artW+borders, l.bodyH,
                            lipgloss.Center, lipgloss.Top, band)
      ```
      Donde `borders` es `2*panelBorder` (narrow) o `3*panelBorder` (medium/wide).
      Usar `lipgloss.PlaceHorizontal` si solo se necesita centrado horizontal y
      el centrado vertical no es crítico en esta iteración.
      Nota: si `PlaceHorizontal` ya resuelve el requisito sin recortar elementos
      obligatorios, es preferible por su menor riesgo. Decidir al aplicar comparando
      el output con el golden 120×30.
- [ ] S2-T3.3 — Verificar `go build ./...` — debe pasar.
- [ ] S2-T3.4 — Verificar `go test ./internal/ui/... -run TestNoLineExceedsWidth`
      — debe pasar en 60, 80, 120 (la introducción de Place no debe crear overflow).

Estimación de líneas modificadas: ~20–30.

---

### S2-T4 — Tests de frontera de classify/computeLayout y assert narrow-no-artwork

**Archivo**: `internal/ui/view_test.go`
**Depende de**: S2-T3 (código de Slice 2 compilado y correcto)
**Mapeo de escenarios**:
- "Narrow breakpoint hides artwork" (@slice2 @edge) — assert directo
- "Breakpoints render distinct deterministic layouts" (@slice2 @happy) — frontera
- Obs-2 del verify-report: falta test unitario de `computeLayout`/`classify`

- [ ] S2-T4.1 — Añadir `TestClassifyBoundaries(t *testing.T)`:
      tabla sobre los valores de frontera exactos del design.md:

      | width | expected breakpoint |
      |-------|-------------------|
      | 59    | bpNarrow          |
      | 60    | bpNarrow          |
      | 89    | bpNarrow          |
      | 90    | bpMedium          |
      | 119   | bpMedium          |
      | 120   | bpWide            |

      Asegurar `classify(width) == expected` para cada caso.

- [ ] S2-T4.2 — Añadir `TestComputeLayoutWidths(t *testing.T)`:
      tabla sobre los mismos 6 anchos con `height=24`. Para cada caso verificar:
      - `l.queueW + l.lyricsW + l.artW <= usable` (no overflow de columnas)
      - `l.queueW >= qMin` y `l.lyricsW >= lMin`
      - En bpNarrow: `l.artW == 0` y `l.showArtwork == false`
      - En bpMedium/bpWide: `l.artW > 0` y `l.showArtwork == true`

- [ ] S2-T4.3 — Añadir `TestComputeLayoutHeight(t *testing.T)`:
      tabla sobre alturas representativas `[20, 24, 30, 40]` con `width=120`:
      - `l.maxQueueRows >= 3` (mínimo siempre ≥ 3)
      - `l.lyricWindow >= 3` (mínimo siempre ≥ 3)
      - `l.lyricWindow % 2 == 1` (siempre impar)
      - A height=20: `l.maxQueueRows < 10` (ventana se reduce con el alto)
      - A height=30: `l.maxQueueRows >= 8` (ventana se expande con el alto)

- [ ] S2-T4.4 — Añadir `Test60x20NarrowNoArtwork(t *testing.T)`:
      Construir modelo con `width=60, height=20`, servicios Artwork y Lyrics activos.
      Llamar `m.View()`, capturar `out`.
      - Asegurar `!strings.Contains(out, "Portada")` — portada no visible en narrow.
      - Asegurar `strings.Contains(out, "Cola")` — cola presente.
      - Asegurar `strings.Contains(out, "Letra")` — letra presente.
      Mapea directamente al escenario "Narrow breakpoint hides artwork".

- [ ] S2-T4.5 — Extender `TestNoLineExceedsWidth` para incluir `width=60` si no
      está ya en la tabla (Slice 1 usa [60, 80, 120]; verificar que el 60 usa
      `height=20` también para cubrir la combinación 60×20 de Slice 2).
      Si ya cubre 60 con height=24, añadir un caso adicional `60×20`.

- [ ] S2-T4.6 — Ejecutar `go test ./internal/ui/... -run TestClassifyBoundaries`
      — debe pasar.
- [ ] S2-T4.7 — Ejecutar `go test ./internal/ui/... -run TestComputeLayoutWidths`
      — debe pasar.
- [ ] S2-T4.8 — Ejecutar `go test ./internal/ui/... -run TestComputeLayoutHeight`
      — debe pasar.
- [ ] S2-T4.9 — Ejecutar `go test ./internal/ui/... -run Test60x20NarrowNoArtwork`
      — debe pasar.

Estimación de líneas añadidas: ~80–100.

---

### S2-T5 — Crear golden 60×20 y regenerar 80×24 / 120×30

**Archivos**: `internal/ui/testdata/view_60x20.golden` (crear),
`view_80x24.golden`, `view_120x30.golden` (regenerar)
**Depende de**: S2-T4 (todos los tests no-golden deben pasar primero)
**Mapeo de escenarios**:
- "Narrow 60×20 golden is locked" (@slice2 @edge)
- "80×24 and 120×30 goldens differ" (@slice1 @edge, mantener)
- "Breakpoints render distinct deterministic layouts" (@slice2 @happy)

- [ ] S2-T5.1 — Añadir el caso `{"60x20", 60, 20}` a la tabla de `TestViewGolden`
      en `view_test.go`.
- [ ] S2-T5.2 — Ejecutar `UPDATE_GOLDEN=1 go test ./internal/ui/... -run TestViewGolden`
      para crear `view_60x20.golden` y regenerar `view_80x24.golden` y
      `view_120x30.golden`.
- [ ] S2-T5.3 — Inspeccionar el diff de los tres goldens:
      - `view_60x20.golden`: confirmar que no contiene "Portada", que contiene "Cola"
        y "Letra", y que ninguna línea excede 60 columnas.
      - `view_80x24.golden`: confirmar que tampoco contiene "Portada" (narrow en 80 cols).
      - `view_120x30.golden`: confirmar que contiene "Portada" (wide, 3 columnas).
      - Los tres goldens deben diferir entre sí.
- [ ] S2-T5.4 — Ejecutar `go test ./internal/ui/...` (suite completa, sin UPDATE_GOLDEN):
      - `TestViewGolden/60x20` — PASS
      - `TestViewGolden/80x24` — PASS
      - `TestViewGolden/120x30` — PASS
      - `TestGoldensDiffer` — PASS (80x24 != 120x30)
      - `TestClassifyBoundaries` — PASS
      - `TestComputeLayoutWidths` — PASS
      - `TestComputeLayoutHeight` — PASS
      - `Test60x20NarrowNoArtwork` — PASS
      - `TestNoLineExceedsWidth` (60/80/120) — PASS
      - `TestStylesNoBackground` — PASS
      - `TestToggleOffParity_*` y demás tests preexistentes — PASS
- [ ] S2-T5.5 — Ejecutar `go build ./...` — debe pasar limpio.
- [ ] S2-T5.6 — Confirmar que ningún archivo fuera del scope fue modificado:
      solo `view.go`, `view_test.go`, `testdata/view_60x20.golden`,
      `testdata/view_80x24.golden`, `testdata/view_120x30.golden`.
      Ningún archivo de `Model`/`Update`/`messages`/`keys`/servicios tocado.
      Ningún código de Slice 3 introducido.
- [ ] S2-T5.7 — Contar líneas cambiadas (excluyendo goldens): confirmar < 400.
      Si se supera 400, aplicar re-slicing 2a (S2-T1+S2-T3, sin 3 columnas) y
      2b (S2-T2+S2-T4+S2-T5).

Estimación: goldens son contenido de fixture, no cuentan hacia el presupuesto.
Adición de un caso en `TestViewGolden`: ~5 líneas.

---

### Tabla resumen de Slice 2

| Tarea | Archivos | Est. líneas | Escenario @slice2 |
|-------|----------|-------------|-------------------|
| S2-T1 Altura dinámica en computeLayout | `view.go` | ~15–20 mod | Vertical space without clipping |
| S2-T2 Narrow 2-col / medium-wide 3-col | `view.go` | ~25–35 mod | Narrow hides artwork, Distinct layouts |
| S2-T3 Place/PlaceVertical sección media | `view.go` | ~20–30 mod | Vertical space without clipping |
| S2-T4 Tests de frontera + narrow-no-artwork | `view_test.go` | ~80–100 add | Narrow hides artwork, Distinct layouts |
| S2-T5 Crear 60×20 golden + regen 80/120 | `testdata/*.golden` + 5 líneas en `view_test.go` | ~5 add + regen | Narrow 60×20 locked, 80≠120 |

**Total líneas de código (view.go + view_test.go)**: ~145–190 líneas cambiadas/añadidas.
**Con margen de implementación real (~20%)**: ~175–230 líneas — dentro del presupuesto de 400.
Los golden files son regenerados/creados, no cuentan como líneas de código autorizadas.

Re-slicing 2a/2b: solo si la implementación real supera 400 líneas. Umbral esperado no
cruzado con la estimación actual.

---

### Verification checklist de Slice 2 (para sign-off de apply)

- [ ] `go build ./...` — green
- [ ] `go vet ./...` — sin hallazgos
- [ ] `go test ./internal/ui/...` — todos los tests pasan
- [ ] `TestClassifyBoundaries` (59/60/89/90/119/120) — PASS
- [ ] `TestComputeLayoutWidths` (6 fronteras) — PASS
- [ ] `TestComputeLayoutHeight` (20/24/30/40 rows) — PASS
- [ ] `Test60x20NarrowNoArtwork` — PASS (sin "Portada", con "Cola" y "Letra")
- [ ] `TestNoLineExceedsWidth` en 60/80/120 — PASS
- [ ] `TestGoldensDiffer` (80x24 != 120x30) — PASS
- [ ] `TestViewGolden/60x20`, `/80x24`, `/120x30` — PASS
- [ ] `TestStylesNoBackground` — PASS (sin regresión)
- [ ] `TestToggleOffParity_*` y demás tests preexistentes — PASS
- [ ] view_60x20.golden: no contiene "Portada"; contiene "Cola" y "Letra"
- [ ] view_120x30.golden: contiene "Portada" (3 columnas)
- [ ] Solo 5 archivos modificados (view.go, view_test.go, 3 goldens); sin Model/Update/keys/messages
- [ ] Ningún código de Slice 3 introducido
- [ ] Líneas de código cambiadas < 400

---

## Slice 3 — Modales, Biblioteca y Deuda

Scope: presentacional; solo se tocan `internal/ui/styles.go`, `internal/ui/view.go`,
`internal/ui/view_test.go`, `internal/ui/update_test.go` (único ajuste de deuda en
`TestRenderQueueWindowsLongQueue`) y `internal/ui/testdata/*.golden` (regeneración
y/o golden nuevo del modal de resultados). No se toca `model.go`, `update.go`,
`messages.go`, `keys.go` ni ningún servicio.

Estimación de líneas de código (sin goldens): ~180–250 (dentro del presupuesto de 400).

Contexto de implementación relevante:
- `picker` y `resultsList` son `list.Model` de `github.com/charmbracelet/bubbles v1.0.0`.
  Ambos usan `list.NewDefaultDelegate()` sin customización. El delegate expone
  `d.Styles` (`DefaultItemStyles`) con campos `NormalTitle`, `NormalDesc`,
  `SelectedTitle`, `SelectedDesc`, `DimmedTitle`, `DimmedDesc`. Ninguno define
  `Background` en los items, pero la barra de título del list (`list.Styles.Title`)
  sí aplica `Background(lipgloss.Color("62"))` vía `DefaultStyles()`.
- `modeResults` renderiza `m.resultsList.View()` directamente sin panel wrapper.
- `modePicker`/`modeLyricsPicker` renderizan `m.picker.View()` directamente.
- `renderLibrary` usa `renderLibList` con estilos `selected` y `dim` del modelo;
  la selección ya distingue por color/negrita/prefijo ➤, sin fondo opaco.
- El test `TestRenderQueueWindowsLongQueue` (update_test.go línea 525) crea el
  modelo con `newTestModel` (width=120, height=40) y llama `m.renderQueue()`.
  A height=40 con el techo actual de 10, `maxQueueRows=10` → "▼ 90 más". Con
  el techo subido a 20 y height=40: `bodyH = max(40-(11+1), 4) = 28`,
  `maxQueueRows = clamp(28-5, 3, 20) = 20` → window=20, "▼ 80 más". El test
  deberá reflejar este nuevo comportamiento.

Las tareas DEBEN ejecutarse en orden numérico. Dependencias indicadas en cada una.

---

### S3-T1 — Aplicar estilos Caelestia al delegate de los modales (modeResults y picker)

**Archivos**: `internal/ui/styles.go`, `internal/ui/view.go`
**Depende de**: nada (primer cambio de Slice 3)
**Mapeo de escenario**: "Modals, library, and pickers preserved and translucent" (@slice3 @happy)

El objetivo es sustituir `list.NewDefaultDelegate()` por un delegate customizado
con estilos Caelestia: foreground/negrita para la selección (mauve o teal), color
muted para estado normal, sin ningún `Background` en ninguna fila. Además, el
`list.Styles.Title` (barra "Resultados") debe perder el `Background("62")` heredado
de `DefaultStyles()`.

- [ ] S3-T1.1 — En `styles.go`, añadir una función `caelestiaListDelegate()` que
      construya y devuelva un `list.DefaultDelegate` con estilos Caelestia:
      ```go
      func caelestiaListDelegate() list.DefaultDelegate {
          d := list.NewDefaultDelegate()
          d.Styles.NormalTitle = lipgloss.NewStyle().
              Foreground(lipgloss.Color("#a0a0a0")).Padding(0, 0, 0, 2)
          d.Styles.NormalDesc = d.Styles.NormalTitle.
              Foreground(lipgloss.Color("#a0a0a0"))
          d.Styles.SelectedTitle = lipgloss.NewStyle().
              Bold(true).
              Foreground(lipgloss.Color("#00f5d4")).
              Border(lipgloss.NormalBorder(), false, false, false, true).
              BorderForeground(lipgloss.Color("#e0aaff")).
              Padding(0, 0, 0, 1)
          d.Styles.SelectedDesc = d.Styles.SelectedTitle.
              Foreground(lipgloss.Color("#e0aaff"))
          d.Styles.DimmedTitle = d.Styles.NormalTitle
          d.Styles.DimmedDesc = d.Styles.NormalDesc
          return d
      }
      ```
      Importante: ninguno de estos estilos puede definir `Background`. El borde
      izquierdo de selección se logra con `Border(NormalBorder(), false,false,false,true)`,
      que añade un `│` lateral sin fondo de fila. Revisar que `GetBackground()` de
      cada subestilo sea `NoColor{}` o `Color("")` antes de confirmar.

- [ ] S3-T1.2 — En `model.go` NO se toca (regla de scope). En su lugar, añadir a
      `view.go` una función `themedResultsList(m Model) list.Model` que tome
      `m.resultsList`, le asigne el delegate de S3-T1.1 y sobreescriba el estilo
      de la barra de título del list para eliminar el `Background`:
      ```go
      func themedList(l list.Model) list.Model {
          l.SetDelegate(caelestiaListDelegate())
          s := l.Styles
          s.Title = lipgloss.NewStyle().
              Bold(true).
              Foreground(lipgloss.Color("#e0aaff")).
              Padding(0, 1)
          l.Styles = s
          return l
      }
      ```
      Esta función actúa sobre una copia del `list.Model` (valor, no puntero) y
      devuelve la copia tematizada. No muta el estado del modelo.

- [ ] S3-T1.3 — En `view.go`, en el bloque `modeResults`, reemplazar:
      ```go
      rb.WriteString(m.resultsList.View())
      ```
      por:
      ```go
      rb.WriteString(themedList(m.resultsList).View())
      ```

- [ ] S3-T1.4 — En `view.go`, en el bloque `modePicker || modeLyricsPicker`,
      reemplazar:
      ```go
      return m.picker.View()
      ```
      por:
      ```go
      return themedList(m.picker).View()
      ```

- [ ] S3-T1.5 — Verificar `go build ./...` — debe pasar sin errores.
- [ ] S3-T1.6 — Verificar `go vet ./...` — sin hallazgos.

Estimación de líneas añadidas/modificadas: ~40–55 (styles.go ~25 add, view.go ~15 mod).

---

### S3-T2 — Deuda: subir techo maxQueueRows de 10 a 20 en computeLayout

**Archivo**: `internal/ui/view.go`
**Depende de**: S3-T1 (compilación verde)
**Mapeo**: deuda documentada en verify-report.md S2-11 Desviación 1 / Obs-S2-3;
  reconcilia design.md D4 ("clamp(bodyH-2, 3, 20)") con lo implementado.

- [ ] S3-T2.1 — En `computeLayout`, localizar la línea:
      ```go
      maxQueueRows := clamp(bodyH-5, 3, 10)
      ```
      Cambiarla a:
      ```go
      maxQueueRows := clamp(bodyH-5, 3, 20)
      ```
      Solo se cambia el techo (10 → 20). El offset `-5` refleja el overhead real
      del panel (borde×2 + encabezado + marcadores ▲/▼) y se mantiene sin cambios
      (es más preciso que el `-2` del spec). El techo 20 es el valor del design D4.

- [ ] S3-T2.2 — Verificar `go build ./...` — debe pasar.
- [ ] S3-T2.3 — Ejecutar `go test ./internal/ui/... -run TestComputeLayoutHeight`
      y confirmar que el test sigue pasando. A height=30: `bodyH=17`,
      `maxQueueRows = clamp(12, 3, 20) = 12 ≥ 8` — PASS.
      A height=40: `bodyH=28`, `maxQueueRows = clamp(23, 3, 20) = 20` — satisface
      el assert "a 30 filas la cola debe expandirse: maxQueueRows ≥ 8".
      A height=20: `bodyH=7`, `maxQueueRows = clamp(2, 3, 20) = 3 < 10` — PASS.

Estimación de líneas modificadas: 1 línea.

---

### S3-T3 — Actualizar TestRenderQueueWindowsLongQueue para el nuevo techo de cola

**Archivo**: `internal/ui/update_test.go`
**Depende de**: S3-T2 (techo de cola subido a 20; el test falla con el valor anterior)
**Mapeo**: deuda documentada en verify-report.md Obs-S2-3; escenario "Queue and
  lyrics behaviors preserved" (@slice2 @happy) preservado con el nuevo comportamiento.

El test usa `newTestModel` (width=120, height=40). Con el techo 20:
`bodyH = max(40-(11+1), 4) = 28`, `maxQueueRows = clamp(23, 3, 20) = 20`.
La cola de 100 ítems mostrará 20 filas → "▼ 80 más" en lugar de "▼ 90 más".
El assert "Track 050 fuera de ventana" se mantiene (50 > 20, correcto).
El assert de crecimiento controlado debe subir de 20 a 30 líneas (20 filas + heading +
marcador ▼ + borde = ~23 líneas; con margen → 30).

- [ ] S3-T3.1 — En `update_test.go`, localizar `TestRenderQueueWindowsLongQueue`
      (línea ~525). Reemplazar el comentario y el assert de "▼ 90 más":
      ```go
      // Desde el inicio (idx 0): se muestran 10 y el resto se indica con el marcador.
      if !strings.Contains(out, "▼ 90 más") {
          t.Fatalf("esperaba marcador '▼ 90 más'; got:\n%s", out)
      }
      ```
      por:
      ```go
      // Desde el inicio (idx 0): con height=40 maxQueueRows=20 se muestran 20
      // filas y el resto se indica con el marcador de desbordamiento.
      if !strings.Contains(out, "▼ 80 más") {
          t.Fatalf("esperaba marcador '▼ 80 más'; got:\n%s", out)
      }
      ```

- [ ] S3-T3.2 — Actualizar el assert de crecimiento controlado en el mismo test.
      Reemplazar:
      ```go
      if n := strings.Count(out, "\n"); n > 20 {
          t.Fatalf("el panel de cola creció demasiado: %d líneas", n)
      }
      ```
      por:
      ```go
      if n := strings.Count(out, "\n"); n > 30 {
          t.Fatalf("el panel de cola creció demasiado: %d líneas", n)
      }
      ```

- [ ] S3-T3.3 — Ejecutar `go test ./internal/ui/... -run TestRenderQueueWindowsLongQueue`
      — debe pasar.

Estimación de líneas modificadas: ~8 líneas.

---

### S3-T4 — Test de paleta hexadecimal de acentos (Obs-1)

**Archivo**: `internal/ui/view_test.go`
**Depende de**: S3-T1 (estilos Caelestia en delegate ya definidos; build verde)
**Mapeo de escenario**: "All colors match Caelestia palette" (@slice1 @happy);
  cierra la brecha Obs-1 del verify-report.md de Slice 1.

- [ ] S3-T4.1 — Añadir `TestCaelestiaAccentColors(t *testing.T)` en `view_test.go`:
      ```go
      func TestCaelestiaAccentColors(t *testing.T) {
          s := defaultStyles()
          cases := []struct {
              name  string
              color lipgloss.Color
          }{
              {"accent mauve (heading/border/viz/errorMsg/selected-border)", "#e0aaff"},
              {"highlight teal (selected/current)", "#00f5d4"},
              {"muted (dim/help)", "#a0a0a0"},
          }
          for _, tc := range cases {
              tc := tc
              t.Run(tc.name, func(t *testing.T) {
                  switch tc.color {
                  case "#e0aaff":
                      if s.heading.GetForeground() != lipgloss.Color("#e0aaff") {
                          t.Errorf("heading foreground no es mauve: %v", s.heading.GetForeground())
                      }
                  case "#00f5d4":
                      if s.selected.GetForeground() != lipgloss.Color("#00f5d4") {
                          t.Errorf("selected foreground no es teal: %v", s.selected.GetForeground())
                      }
                      if s.current.GetForeground() != lipgloss.Color("#00f5d4") {
                          t.Errorf("current foreground no es teal: %v", s.current.GetForeground())
                      }
                  case "#a0a0a0":
                      if s.dim.GetForeground() != lipgloss.Color("#a0a0a0") {
                          t.Errorf("dim foreground no es muted: %v", s.dim.GetForeground())
                      }
                      if s.help.GetForeground() != lipgloss.Color("#a0a0a0") {
                          t.Errorf("help foreground no es muted: %v", s.help.GetForeground())
                      }
                  }
              })
          }
      }
      ```
      El test afirma los 3 colores hexadecimales por nombre sobre los estilos de
      `defaultStyles()` — cobertura directa e independiente de goldens.

- [ ] S3-T4.2 — Ejecutar `go test ./internal/ui/... -run TestCaelestiaAccentColors`
      — debe pasar.

Estimación de líneas añadidas: ~40–50.

---

### S3-T5 — Extender assert no-Background a los estilos del delegate modal

**Archivo**: `internal/ui/view_test.go`
**Depende de**: S3-T1 (delegate ya existe), S3-T4 (build verde con nuevos tests)
**Mapeo de escenario**: "Modals, library, and pickers preserved and translucent"
  (@slice3 @happy); extiende la Decision 2 del design al scope de modales.

- [ ] S3-T5.1 — Añadir `TestDelegateNoBackground(t *testing.T)` en `view_test.go`:
      ```go
      func TestDelegateNoBackground(t *testing.T) {
          d := caelestiaListDelegate()
          checks := []struct {
              name  string
              style lipgloss.Style
          }{
              {"NormalTitle", d.Styles.NormalTitle},
              {"NormalDesc", d.Styles.NormalDesc},
              {"SelectedTitle", d.Styles.SelectedTitle},
              {"SelectedDesc", d.Styles.SelectedDesc},
              {"DimmedTitle", d.Styles.DimmedTitle},
              {"DimmedDesc", d.Styles.DimmedDesc},
          }
          for _, c := range checks {
              c := c
              t.Run(c.name, func(t *testing.T) {
                  if !hasNoBackground(c.style) {
                      t.Errorf("delegate.%s no debe definir Background; got %#v",
                          c.name, c.style.GetBackground())
                  }
              })
          }
      }
      ```
      Reutiliza el helper `hasNoBackground` definido en Slice 1.

- [ ] S3-T5.2 — Ejecutar `go test ./internal/ui/... -run TestDelegateNoBackground`
      — debe pasar.

Estimación de líneas añadidas: ~30.

---

### S3-T6 — Verificar paridad: renderLibrary ya es translúcida

**Archivo**: `internal/ui/view_test.go`
**Depende de**: S3-T5 (suite verde)
**Mapeo de escenario**: "Modals, library, and pickers preserved and translucent"
  (@slice3 @happy), rama library.

La inspección del código muestra que `renderLibrary` usa `m.styles.selected`
(teal foreground + bold, sin Background) y `m.styles.dim` (muted foreground, sin
Background). La selección se distingue por prefijo ➤ y color — no por relleno.
No se requiere cambio de código; solo una verificación de test.

- [ ] S3-T6.1 — Añadir `TestLibraryViewIsTranslucent(t *testing.T)` en `view_test.go`:
      construir un modelo con `modeLibrary`, `libSection = sectionFavorites` y al
      menos 2 tracks en `libFavorites`; llamar `m.View()`; afirmar que el output
      contiene los títulos de las pistas y que el cursor ➤ está presente. (No es
      posible afirmar ausencia de Background en ANSI con goldens en plaintext, pero
      se puede afirmar que el markup de selección no incluye relleno de fondo
      inspeccionando el estilo directamente con `hasNoBackground(m.styles.selected)`
      y `hasNoBackground(m.styles.dim)`.)
      ```go
      func TestLibraryViewIsTranslucent(t *testing.T) {
          m := newTestModel(t, Services{})
          m.mode = modeLibrary
          m.libSection = sectionFavorites
          m.libFavorites = []search.Result{
              {ID: "a", Title: "Canción A", Uploader: "Artista A"},
              {ID: "b", Title: "Canción B", Uploader: "Artista B"},
          }
          m.libCursor = 0
          out := m.View()
          if !strings.Contains(out, "Canción A") {
              t.Errorf("biblioteca debe mostrar los ítems; got:\n%s", out)
          }
          if !strings.Contains(out, "➤") {
              t.Errorf("biblioteca debe mostrar el cursor ➤; got:\n%s", out)
          }
          // Verificar que los estilos de selección no tienen Background.
          if !hasNoBackground(m.styles.selected) {
              t.Errorf("styles.selected no debe definir Background")
          }
          if !hasNoBackground(m.styles.dim) {
              t.Errorf("styles.dim no debe definir Background")
          }
      }
      ```

- [ ] S3-T6.2 — Ejecutar `go test ./internal/ui/... -run TestLibraryViewIsTranslucent`
      — debe pasar.

Estimación de líneas añadidas: ~25–30.

---

### S3-T7 — Regenerar goldens afectados y añadir golden del modal de resultados

**Archivos**: `internal/ui/testdata/*.golden` (regen o nuevo),
  `internal/ui/view_test.go` (añadir caso de golden modal)
**Depende de**: S3-T6 (suite no-golden completamente verde)
**Mapeo de escenario**: "Modals, library, and pickers preserved and translucent"
  (@slice3 @happy); "Golden Determinism" (@slice1 @slice2).

Los cambios de S3-T1 (delegate + título del list) alteran el output de
`m.resultsList.View()`. Aunque los goldens existentes (60×20, 80×24, 120×30)
no cubren `modeResults` (usan `modeNormal`), es necesario crear un golden del
modal para bloquearlo y detectar regresiones futuras.

- [ ] S3-T7.1 — En `view_test.go`, añadir el caso `{"results_120x30", modeResults, 120, 30}`
      a una nueva función `TestResultsModalGolden` que construya un modelo en
      `modeResults`, popule `m.resultsList` con 5 ítems representativos y llame
      `m.View()`. Usar `compareGolden` al archivo
      `testdata/view_results_120x30.golden`.
      ```go
      func TestResultsModalGolden(t *testing.T) {
          m := newTestModel(t, Services{})
          m.mode = modeResults
          m.width, m.height = 120, 30
          items := []list.Item{
              resultItem{r: search.Result{ID: "a", Title: "Canción A", Uploader: "Artista A"}},
              resultItem{r: search.Result{ID: "b", Title: "Canción B", Uploader: "Artista B"}},
              resultItem{r: search.Result{ID: "c", Title: "Canción C", Uploader: "Artista C"}},
              resultItem{r: search.Result{ID: "d", Title: "Canción D", Uploader: "Artista D"}},
              resultItem{r: search.Result{ID: "e", Title: "Canción E", Uploader: "Artista E"}},
          }
          m.resultsList.SetItems(items)
          m.resultsList = themedList(m.resultsList)
          out := m.View()
          compareGolden(t, filepath.Join("testdata", "view_results_120x30.golden"), out)
      }
      ```
      Nota: `themedList` se llama explícitamente aquí para que el test ejerza el
      mismo path que `View()` en modeResults.

- [ ] S3-T7.2 — Verificar si los goldens existentes (60×20, 80×24, 120×30) son
      afectados por los cambios de S3-T1. Estos goldens usan `modeNormal`, que no
      pasa por `themedList`. Si `computeLayout` y los render helpers de la vista
      principal no cambiaron, los goldens no deben cambiar. Confirmar ejecutando:
      `go test ./internal/ui/... -run TestViewGolden` — si falla, regenerar.

- [ ] S3-T7.3 — Crear el golden del modal ejecutando:
      `UPDATE_GOLDEN=1 go test ./internal/ui/... -run TestResultsModalGolden`

- [ ] S3-T7.4 — Inspeccionar `view_results_120x30.golden`:
      - Confirmar que contiene "Resultados" (título del modal).
      - Confirmar que contiene "Canción A", "Artista A" (ítems).
      - Confirmar que contiene la línea de ayuda del modal
        ("enter encolar · a +playlist · f favorito · ↑/↓ navegar · esc cerrar").
      - Confirmar que ninguna línea excede 120 columnas.

- [ ] S3-T7.5 — Ejecutar `go test ./internal/ui/...` (suite completa, sin UPDATE_GOLDEN)
      — todos los tests deben pasar.
- [ ] S3-T7.6 — Ejecutar `go build ./...` — debe pasar limpio.

Estimación: golden es fixture no contado hacia presupuesto; ~30 líneas en view_test.go.

---

### S3-T8 — Reconciliar design.md D4 con la implementación real

**Archivo**: `internal/ui/view.go` (comentario en `computeLayout`)
**Depende de**: S3-T2 (techo ya subido a 20)
**Mapeo**: nota de reconciliación del alcance de Slice 3.

El `design.md` D4 dice `maxQueueRows = clamp(bodyH-2 (heading+borders), 3, 20)`.
La implementación usa `clamp(bodyH-5, 3, 20)` (con -5 para descontar heading,
borde×2 y marcadores ▲/▼). El techo ya es 20. Solo es necesario actualizar el
comentario del código para que sea consistente con lo implementado.

- [ ] S3-T8.1 — En `computeLayout`, localizar el comentario sobre `maxQueueRows`:
      ```go
      // Filas de la cola: bodyH menos el chrome real del panel (2 de borde,
      // 1 de encabezado y hasta 2 marcadores ▲/▼), con piso 3 para no colapsar
      // en alturas chicas. El techo queda en 10 (la ventana histórica): la letra
      // es la región que crece primero con el alto.
      maxQueueRows := clamp(bodyH-5, 3, 10)
      ```
      Actualizar el comentario (el código ya fue cambiado en S3-T2):
      ```go
      // Filas de la cola: bodyH menos el chrome real del panel (2 de borde,
      // 1 de encabezado y hasta 2 marcadores ▲/▼ = 5 filas de overhead), con
      // piso 3 para no colapsar en alturas chicas y techo 20 (design D4).
      maxQueueRows := clamp(bodyH-5, 3, 20)
      ```

Estimación de líneas modificadas: ~4 (solo comentario + la línea de código cambiada en S3-T2).

---

### S3-T9 — Verificación final de la Slice 3

**Depende de**: S3-T8 (todos los cambios de Slice 3 aplicados)

- [ ] S3-T9.1 — Ejecutar `go build ./...` — debe pasar limpio.
- [ ] S3-T9.2 — Ejecutar `go vet ./...` — sin hallazgos.
- [ ] S3-T9.3 — Ejecutar `go test ./internal/ui/...` — todos los tests pasan,
      incluyendo:
      - `TestCaelestiaAccentColors` — PASS (cierra Obs-1).
      - `TestDelegateNoBackground` (todos los subestilos) — PASS.
      - `TestLibraryViewIsTranslucent` — PASS.
      - `TestResultsModalGolden` — PASS.
      - `TestRenderQueueWindowsLongQueue` (▼ 80 más) — PASS (deuda cerrada).
      - `TestComputeLayoutHeight` (todos los casos) — PASS.
      - `TestStylesNoBackground` — PASS (sin regresión Slice 1).
      - `TestGoldensDiffer` (3 pares) — PASS.
      - `TestViewGolden/60x20`, `/80x24`, `/120x30` — PASS.
      - `TestNoLineExceedsWidth` (todos los tamaños) — PASS.
      - `TestToggleOffParity_*` — PASS.
- [ ] S3-T9.4 — Ejecutar `go test ./...` — 17 paquetes, todos limpios.
- [ ] S3-T9.5 — Confirmar scope: solo se tocaron los archivos autorizados:
      `styles.go`, `view.go`, `view_test.go`, `update_test.go`,
      `testdata/view_results_120x30.golden` (y, si regresaron, los 3 goldens
      existentes). No se tocó `model.go`, `update.go`, `messages.go`, `keys.go`
      ni ningún servicio.
- [ ] S3-T9.6 — Contar líneas cambiadas (excluyendo goldens): confirmar < 400.
      Estimación: ~175–230 líneas. Si se supera 400, identificar el bloque más
      grande (probablemente S3-T4/S3-T5/S3-T6 en view_test.go) y dividirlo en
      una sub-slice.

---

### Tabla resumen de Slice 3

| Tarea | Archivos | Est. líneas | Escenario / Deuda |
|-------|----------|-------------|-------------------|
| S3-T1 Delegate Caelestia para modales | `styles.go`, `view.go` | ~50–55 add/mod | @slice3 "Modals… translucent" |
| S3-T2 Subir techo maxQueueRows 10→20 | `view.go` | ~1 mod | Deuda S2 / Design D4 |
| S3-T3 Actualizar TestRenderQueueWindowsLongQueue | `update_test.go` | ~8 mod | Deuda S2 / @slice2 queue preserved |
| S3-T4 Test paleta hexadecimal (Obs-1) | `view_test.go` | ~45–50 add | @slice1 "All colors match palette" |
| S3-T5 Extender no-Background a delegate | `view_test.go` | ~30 add | @slice3 "no delegate row opaque bg" |
| S3-T6 Verificar library translúcida | `view_test.go` | ~28 add | @slice3 "library translucent" |
| S3-T7 Golden modal resultados | `view_test.go` + `testdata/*.golden` | ~30 add + regen | @slice3 Golden del modal |
| S3-T8 Reconciliar comentario D4 | `view.go` | ~4 mod | Reconciliación design.md D4 |
| S3-T9 Verificación final | — | — | Checklist de cierre |

**Total líneas de código (styles + view + view_test + update_test)**: ~180–250
cambiadas/añadidas (< 400). Los golden files son fixture, no cuentan.

---

### Verification checklist de Slice 3 (para sign-off de apply)

- [ ] `go build ./...` — green
- [ ] `go vet ./...` — sin hallazgos
- [ ] `go test ./internal/ui/...` — todos los tests pasan
- [ ] `go test ./...` — 17 paquetes limpios
- [ ] `TestCaelestiaAccentColors` — PASS (hex #e0aaff, #00f5d4, #a0a0a0 por nombre)
- [ ] `TestDelegateNoBackground` (6 subestilos) — PASS (ningún Background en delegate)
- [ ] `TestLibraryViewIsTranslucent` — PASS (ítems + cursor ➤ presentes; estilos sin Background)
- [ ] `TestResultsModalGolden/results_120x30` — PASS (golden bloqueado)
- [ ] `TestRenderQueueWindowsLongQueue` — PASS (▼ 80 más, techo 20)
- [ ] `TestComputeLayoutHeight` — PASS (sin regresión)
- [ ] `TestStylesNoBackground` — PASS (sin regresión Slice 1)
- [ ] `TestGoldensDiffer` — PASS (sin regresión)
- [ ] `TestViewGolden/60x20`, `/80x24`, `/120x30` — PASS (sin regresión)
- [ ] `TestNoLineExceedsWidth` — PASS (sin regresión)
- [ ] `TestToggleOffParity_*` — PASS (sin regresión)
- [ ] `TestRenderQueueWindowsLongQueue` (▼ 80 más) — PASS
- [ ] `caelestiaListDelegate()` en styles.go: ningún subestilo tiene Background
- [ ] `list.Styles.Title` en themedList: sin Background, foreground mauve #e0aaff
- [ ] La selección en delegate distingue por color/negrita/borde-izquierdo, no por relleno opaco
- [ ] Scope: solo 4 archivos fuente tocados (styles.go, view.go, view_test.go, update_test.go)
      más testdata/view_results_120x30.golden; sin model.go/update.go/messages.go/keys.go/servicios
- [ ] Líneas de código cambiadas < 400
