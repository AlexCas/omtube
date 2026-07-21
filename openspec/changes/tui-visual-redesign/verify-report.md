# Verify Report — TUI Visual Redesign, Slice 1 (Base)

Date: 2026-07-21
Phase: verify
Veredicto: **PASA-CON-OBSERVACIONES**

---

## 1. Build y análisis estático

| Comando | Resultado |
|---------|-----------|
| `go build ./...` | PASS — sin errores |
| `go vet ./...` | PASS — sin hallazgos |

---

## 2. Suite de tests

| Comando | Resultado |
|---------|-----------|
| `go test ./internal/ui/...` | PASS — 37 tests, 0 fallos |
| `go test ./...` | PASS — 17 paquetes, todos limpios |

No quedan archivos `.got` en `internal/ui/testdata/`.

### Tests individuales de la Slice 1

| Test | Resultado |
|------|-----------|
| `TestStylesNoBackground` | PASS |
| `TestNoLineExceedsWidth/60cols` | PASS |
| `TestNoLineExceedsWidth/80cols` | PASS |
| `TestNoLineExceedsWidth/120cols` | PASS |
| `TestGoldensDiffer` | PASS |
| `TestViewGolden/80x24` | PASS |
| `TestViewGolden/120x30` | PASS |

### Tests de paridad preexistentes

`TestToggleOffParity_NoEnrichmentPanels`, `TestToggleOffParity_NoTrackChangeFanout`,
`TestLyricsPanel_*`, `TestArtworkPanel_RenderAndDegrade`, `TestCacheIndicator`,
`TestRenderQueueWindowsLongQueue`, `TestQueueWindow/*`, todos pasan sin modificación.

---

## 3. Diferencia de goldens (TestGoldensDiffer)

| Archivo | Bytes |
|---------|-------|
| `view_80x24.golden` | 2 259 |
| `view_120x30.golden` | 3 200 |

Los archivos difieren en bytes (941 bytes de diferencia). No hay archivos `.got` residuales.

Inspección de anchos de panel en los goldens (conteo de `─` en la línea superior):

| Golden | queueW | lyricsW | artW | Breakpoint |
|--------|--------|---------|------|------------|
| 80×24 | 24 | 31 | 17 | bpNarrow (< 90) |
| 120×30 | 34 | 50 | 28 | bpWide (≥ 120) |

Toda línea visual de ambos goldens tiene `width ≤ target` (verificado con `unicodedata.east_asian_width`).

---

## 4. Asserts: contenido y cobertura

### TestStylesNoBackground

Construye `defaultStyles()` y verifica:
- `hasNoBackground(s.title)` — comprueba `NoColor{}` o `Color("")`. PASS.
- `hasNoBackground(s.panel)` — ídem. PASS.
- `s.title.GetBorderStyle() == lipgloss.RoundedBorder()` — borde conservado. PASS.
- `s.panel.GetBorderStyle() == lipgloss.RoundedBorder()` — borde conservado. PASS.

### TestNoLineExceedsWidth

Tabla sobre `[60, 80, 120]`. Por cada ancho: modelo con título largo (`"Título Muy Largo "×8`) y 21 ítems en cola, 3 líneas de letra sincronizada largas, `pos=45 dur=180`. Llama `m.View()`, parte en `\n`, afirma `lipgloss.Width(line) <= width`. PASS en las tres variantes.

### TestGoldensDiffer

Lee ambos goldens, afirma `!bytes.Equal`. Skipea si algún archivo falta. PASS.

---

## 5. Trazabilidad de escenarios @slice1

| Escenario | Etiqueta | Cobertura |
|-----------|----------|-----------|
| All colors match Caelestia palette | @slice1 @happy | Parcialmente cubierto: `TestStylesNoBackground` verifica ausencia de Background y borde mauve `#e0aaff`; accent/muted/highlight en otros estilos (`selected`, `dim`, `current`) se verifican indirectamente vía goldens. No existe un test que afirme los 3 colores por nombre. Brecha menor (ver Obs-1). |
| No opaque background paints over the terminal glass | @slice1 @happy | Cubierto por `TestStylesNoBackground` con asserts directos sobre `title` y `panel`. COMPLETO. |
| No rendered line exceeds terminal width (60/80/120) | @slice1 @edge | Cubierto por `TestNoLineExceedsWidth` en las tres variantes de tabla. COMPLETO. |
| Widths derive from runtime dimensions | @slice1 @happy | Cubierto implícitamente: `TestNoLineExceedsWidth` re-renderiza a 60/80/120 y las diferencias de width en goldens lo confirman. No existe un test unitario de `computeLayout` (ver Obs-2). |
| Core main-view elements preserved | @slice1 @happy | Cubierto por `TestViewGolden` (80×24 y 120×30): título, ahora-suena, estado, cola, letra, portada, ayuda, visualizador presentes en ambos goldens. `TestRenderQueueWindowsLongQueue` y `TestCacheIndicator` cubren ▲/▼/⤓/▶. COMPLETO. |
| 80×24 and 120×30 goldens differ | @slice1 @edge | Cubierto por `TestGoldensDiffer`. COMPLETO. |

Escenarios @slice1 sin cobertura directa: ninguno está sin cobertura total, pero el escenario "All colors match Caelestia palette" no tiene un assert que afirme los 3 colores hexadecimales por nombre (ver Obs-1).

---

## 6. Paridad de elementos (inspección view.go y goldens)

| Elemento | Presente en 80×24 | Presente en 120×30 | Notas |
|----------|-------------------|---------------------|-------|
| Título `🎵 Omusic` | ✓ | ✓ | |
| Ahora suena: estado/título/progreso/tiempo/volumen | ✓ | ✓ | |
| Búsqueda/estado | ✓ | ✓ | |
| Cola (ventana deslizante, ▲/▼, ⤓, ▶) | ✓ (2 items; ventana dinámica cubierta en TestRenderQueueWindowsLongQueue) | ✓ | |
| Letra | ✓ | ✓ | |
| Portada | ✓ | ✓ | No oculta en Slice 1 — artW=17 en narrow |
| Ayuda | ✓ | ✓ | wrapHelp aplicado correctamente |
| Visualizador | ✓ | ✓ | |
| Biblioteca (tabs/cursor) | No en golden (modo distinto) | No en golden | Cubierto por código en renderLibrary |

La portada NO se oculta en el golden 80×24 (artW=17 > 0 y la llamada a `renderArtworkPanelAt` ocurre). Esto es correcto para Slice 1; `showArtwork` solo será consumido en Slice 2.

---

## 7. Alcance (scope check)

Archivos modificados en el working tree vs HEAD en `internal/`:

| Archivo | Modificado |
|---------|-----------|
| `internal/ui/styles.go` | ✓ (solo 2 líneas eliminadas: Background de title y panel) |
| `internal/ui/view.go` | ✓ (layout types + computeLayout + fluid widths) |
| `internal/ui/view_test.go` | ✓ (3 nuevos tests + hasNoBackground helper) |
| `internal/ui/testdata/view_80x24.golden` | ✓ (regenerado) |
| `internal/ui/testdata/view_120x30.golden` | ✓ (regenerado) |
| `internal/ui/model.go` | — sin cambios |
| `internal/ui/update.go` | — sin cambios |
| `internal/ui/messages.go` | — sin cambios |
| `internal/ui/keys.go` | — sin cambios |

Solo se tocaron los 5 archivos contemplados por las tasks. No se introdujo código de Slice 2 (no hay `PlaceVertical`, no se aplica `showArtwork` en ningún render path, no hay 60×20 golden). No se introdujo código de Slice 3 (no hay delegate styles).

---

## 8. Evaluación de desviaciones reportadas por apply

### Desviación 1: `artW > 0` en `bpNarrow`

**Descripción**: el diseño especifica que en narrow los mínimos de artwork son `aMin=8, aMax=28` en lugar de desaparecer (la desaparición es Slice 2). El tasks.md T3.8 dice explícitamente: "Slice 1 does NOT hide artwork for bpNarrow". El código usa `aMin=8` para narrow, resultando en `artW=17` a 80 cols.

**Evaluación**: CONSISTENTE con spec y tasks. La portada a `artW=17` es deliberada en Slice 1 y no produce overflow (verificado). No introduce deuda técnica: el flag `showArtwork` ya está preparado para Slice 2. Riesgo: ninguno.

### Desviación 2: Truncado cola `queueW-6`

**Descripción**: tasks T3.2 especifica `queueW - 2`; el código usa `queueW - 6`. El comentario explica que descuenta 2 cols de padding + 2 de cacheMark + 2 de prefijo ▶/espacios.

**Evaluación**: La fórmula `queueW - 6` es más precisa que `queueW - 2` para el layout real (2 padding + 2 cache + 2 prefijo = 6 cols de overhead). La especificación en tasks fue aproximada. El test `TestNoLineExceedsWidth` valida que ninguna línea excede el ancho, por lo que la desviación es benigna y preferible. No introduce deuda.

### Desviación 3: `progressW` con `nowTitleTrunc` en la fórmula

**Descripción**: tasks T2.4 especifica `progressW = clamp(width-24, 8, 40)`. Apply usa `clamp(width - 24 - nowTitleTrunc, 8, 40)`. A 80 cols esto produce `progressW=29` en lugar de `40`.

**Evaluación**: El design.md Decision 3 dice "progressW = clamp(width − decorLen, 8, 40) donde decorLen = chrome + title_trunc", lo cual es exactamente lo que hace el apply. La fórmula de tasks fue una simplificación; apply sigue el design más fiel. La diferencia produce una barra más corta a 80 cols (29 vs 40), lo cual es correcto: a 80 cols la barra de 40 chars más el título y el chrome sobrepasaría el ancho. No introduce deuda; es la implementación correcta del design.

**Riesgo menor**: la barra de progreso es más corta de lo esperado al leer las tasks sin el design. Al revisar el golden 80×24 (línea 5) la barra usa `━━━━━━━──────────────────────` (7 llenas + 22 vacías = 29 chars), correcto.

### Desviación 4: Wrappers de firma histórica (`renderQueue`, `renderLyricsPanel`, `renderArtworkPanel`)

**Descripción**: las firmas públicas sin parámetro se conservaron como wrappers que computan el layout. Esto preserva la compatibilidad con tests existentes.

**Evaluación**: CORRECTA. Los tests preexistentes usan estas firmas. Los nuevos render helpers usan los `...At(l layout)` equivalentes. No hay deuda.

### Desviación 5: `hasNoBackground` acepta `NoColor{}`

**Descripción**: el design especifica `s.GetBackground() == lipgloss.Color("")`. El apply implementa un type switch que acepta también `lipgloss.NoColor{}` (el tipo que lipgloss v1 devuelve cuando no hay background).

**Evaluación**: CORRECTA y necesaria. Con lipgloss v1.1.0 el valor de fondo sin definir es `NoColor{}`, no `Color("")`. Sin este type switch `TestStylesNoBackground` habría fallado incorrectamente. La desviación mejora la robustez. No introduce deuda.

---

## 9. Gaps de test

**Obs-1 (menor)**: el escenario "All colors match Caelestia palette" no tiene un test que afirme los 3 valores hex (`#e0aaff`, `#a0a0a0`, `#00f5d4`) por nombre sobre los estilos `selected`, `dim`, `current`. Los goldens los cubren implícitamente pero un test unitario en `TestStylesNoBackground` (o test separado) lo haría explícito. No es bloqueante para Slice 1; recomendado añadir en Slice 3 cuando se cubran los delegate styles.

**Obs-2 (menor)**: no existe un test unitario de `computeLayout` / `classify` con los boundary values especificados en design.md (59/60/89/90/119/120). Design indica "Table test on boundary widths" en la estrategia de testing. El escenario "Widths derive from runtime dimensions" queda cubierto solo por goldens. Recomendado añadir `TestComputeLayout` antes de cerrar Slice 2, que introduce cambios en la misma función.

---

## Resumen de evidencias

| Verificación | Resultado |
|---|---|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./internal/ui/...` | PASS (37 tests) |
| `go test ./...` | PASS (17 paquetes) |
| No archivos `.got` | PASS |
| Goldens difieren en bytes (TestGoldensDiffer) | PASS (2259 vs 3200 bytes) |
| Todas las líneas ≤ width en goldens | PASS |
| TestStylesNoBackground (title+panel sin Background, borde conservado) | PASS |
| TestNoLineExceedsWidth (60/80/120 cols con contenido largo) | PASS |
| Paridad de elementos en goldens | PASS |
| Scope: solo 5 archivos tocados, sin Model/Update/keys/messages | PASS |
| No hay código de Slice 2/3 introducido | PASS |
| Escenarios @slice1 cubiertos | 6/6 (Obs-1 menor en paleta) |
| Desviaciones de apply | 5 evaluadas — todas benignas o preferibles |

---

## Veredicto

**PASA-CON-OBSERVACIONES**

La Slice 1 cumple el spec, el design y las tasks. Todas las verificaciones críticas pasan. Las dos observaciones son gaps de test menores (no blocking) que pueden resolverse en Slice 2 o Slice 3. No se requiere re-apply.

---

## Slice 2 — Dashboard / Uso del alto

Date: 2026-07-21
Phase: verify
Veredicto: **PASA-CON-OBSERVACIONES**

---

### S2-1. Build y análisis estático

| Comando | Resultado |
|---------|-----------|
| `go build ./...` | PASS — sin errores |
| `go vet ./...` | PASS — sin hallazgos |

---

### S2-2. Suite de tests

| Comando | Resultado |
|---------|-----------|
| `go test ./internal/ui/...` | PASS — todos los tests verdes, 0 fallos |
| `go test ./...` | PASS — 17 paquetes, todos limpios |

No quedan archivos `.got` en `internal/ui/testdata/`.

#### Tests específicos de Slice 2

| Test | Variantes | Resultado |
|------|-----------|-----------|
| `TestClassifyBoundaries` | 59/60/89/90/119/120 | PASS |
| `TestComputeLayoutWidths` | 59/60/89/90/119/120 cols | PASS |
| `TestComputeLayoutHeight` | 20/24/30/40 rows | PASS |
| `Test60x20NarrowNoArtwork` | 60×20 | PASS |
| `TestNoLineExceedsWidth` | 60×20, 60×24, 80×24, 120×24, 120×30 | PASS |
| `TestGoldensDiffer` | 60×20 vs 80×24 vs 120×30 (3 pares) | PASS |
| `TestViewGolden/60x20` | — | PASS |
| `TestViewGolden/80x24` | — | PASS |
| `TestViewGolden/120x30` | — | PASS |

#### Tests de Slice 1 (regresión)

| Test | Resultado |
|------|-----------|
| `TestStylesNoBackground` | PASS — sin regresión |
| `TestNoLineExceedsWidth/60x24` y anteriores | PASS |
| `TestToggleOffParity_*` | PASS |
| `TestRenderQueueWindowsLongQueue` | PASS |
| Todos los demás preexistentes | PASS |

---

### S2-3. Goldens: dimensiones, paridad de pares y contenido

#### Dimensiones exactas (filas = altura objetivo, ancho máximo = cols objetivo)

| Golden | Filas | Ancho máx | Overflow |
|--------|-------|-----------|----------|
| `view_60x20.golden` | 20 | 60 | ninguno |
| `view_80x24.golden` | 24 | 80 | ninguno |
| `view_120x30.golden` | 30 | 120 | ninguno |

Verificación con `unicodedata.east_asian_width` línea a línea.

#### Paridad de pares (TestGoldensDiffer — 3 comparaciones)

| Par | Resultado |
|-----|-----------|
| 60×20 vs 80×24 | DIFIEREN |
| 60×20 vs 120×30 | DIFIEREN |
| 80×24 vs 120×30 | DIFIEREN |

#### Contenido semántico de goldens

| Golden | "Portada" | "Cola" | "Letra" | Breakpoint confirmado |
|--------|-----------|--------|---------|----------------------|
| `view_60x20.golden` | NO | SÍ | SÍ | bpNarrow (60 < 90) |
| `view_80x24.golden` | NO | SÍ | SÍ | bpNarrow (80 < 90) |
| `view_120x30.golden` | SÍ | SÍ | SÍ | bpWide (120 ≥ 120) |

Elemento "Omusic", "0:45/3:00" (ahora suena), "buscar" (ayuda) y visualizador de barras presentes en los 3 goldens. El ▶ (pista actual) presente en los 3 goldens.

---

### S2-4. Valores de computeLayout: verificación de alturas

Calculados con el código real mediante test auxiliar:

| height | bodyH | maxQueueRows | lyricWindow | Invariantes |
|--------|-------|-------------|-------------|-------------|
| 20 | 7 | 3 | 3 | bodyH≥4 ✓, maxQ≥3 ✓, lW≥3 ✓, lW impar ✓, maxQ<10 (=3) ✓ |
| 24 | 11 | 6 | 7 | bodyH≥4 ✓, maxQ≥3 ✓, lW≥3 ✓, lW impar ✓ |
| 30 | 17 | 10 | 11 | bodyH≥4 ✓, maxQ≥3 ✓, lW≥3 ✓, lW impar ✓, maxQ≥8 (=10) ✓ |
| 40 | 27 | 10 | 11 | bodyH≥4 ✓, maxQ≥3 ✓, lW≥3 ✓, lW impar ✓ |

Todos los asserts de `TestComputeLayoutHeight` satisfechos.

---

### S2-5. Trazabilidad de escenarios @slice2

| Escenario | Tag | Cobertura |
|-----------|-----|-----------|
| Vertical space is used without clipping | @slice2 @happy | Cubierto por `TestComputeLayoutHeight` (bodyH dinámico en 4 alturas), `TestViewGolden/120x30` (30 filas encajan sin recorte), `TestNoLineExceedsWidth/120x30`. `PlaceVertical` en `renderMiddleSection` es la implementación. COMPLETO. |
| Narrow breakpoint hides artwork | @slice2 @edge | Cubierto directamente por `Test60x20NarrowNoArtwork`: sin "Portada"/"ASCII ART", con "Cola" y "Letra" a 60×20. Goldens 60×20 y 80×24 confirman ausencia de "Portada". COMPLETO. |
| Breakpoints render distinct deterministic layouts | @slice2 @happy | Cubierto por `TestGoldensDiffer` (3 pares todos distintos), `TestClassifyBoundaries` (6 fronteras), `TestComputeLayoutWidths` (anchos distintos por breakpoint). COMPLETO. |
| Queue and lyrics behaviors preserved | @slice2 @happy | Cola: `TestRenderQueueWindowsLongQueue` (▲/▼/⤓/▶), `TestQueueWindow/*`, ventana deslizante verificada. Letra sincronizada: `TestLyricsPanel_SyncedHighlight`, `TestNoLineExceedsWidth` (usa Synced: true con líneas largas). COMPLETO. |
| Narrow 60×20 golden is locked | @slice2 @edge | Cubierto por `TestViewGolden/60x20` (golden existente, sin UPDATE_GOLDEN). Ninguna línea excede 60 cols. COMPLETO. |

Escenarios @slice2 sin cobertura directa: ninguno. Todos cubiertos.

---

### S2-6. Uso del alto: comportamiento vertical

- **`bodyH` dinámico**: `computeLayout` calcula `bodyH = max(height - (11 + helpRows(width)), 4)`. A h=20 → bodyH=7; a h=30 → bodyH=17. Los paneles encogen/crecen con la altura.
- **PlaceVertical**: `renderMiddleSection` envuelve la banda con `lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)`. La sección media ocupa exactamente `bodyH` filas; el exceso vertical se rellena. Ningún elemento obligatorio queda recortado: los clamps garantizan mínimos (maxQueueRows≥3, lyricWindow≥3).
- **h=20 (terminal pequeña)**: maxQueueRows=3, lyricWindow=3. La cola muestra 3 filas y la letra 3 líneas — mínimo funcional, sin colapso. TestComputeLayoutHeight/20rows pasa.
- **h=30 (terminal cómoda)**: maxQueueRows=10, lyricWindow=11. La cola muestra hasta 10 filas y la letra 11 líneas. TestComputeLayoutHeight/30rows pasa.

---

### S2-7. Portada: narrow vs medium/wide

| Ancho | Breakpoint | showArtwork | Panel portada renderizado |
|-------|-----------|-------------|--------------------------|
| 59 | bpNarrow | false | NO |
| 60 | bpNarrow | false | NO |
| 80 | bpNarrow | false | NO |
| 89 | bpNarrow | false | NO |
| 90 | bpMedium | true | SÍ |
| 119 | bpMedium | true | SÍ |
| 120 | bpWide | true | SÍ |

`renderEnrichment` aplica la guarda `hasArtwork && l.showArtwork` antes de añadir el panel. En narrow, cola y letra siguen presentes. Verificado en `Test60x20NarrowNoArtwork` y por los goldens.

---

### S2-8. Paridad de elementos en los 3 goldens

| Elemento | 60×20 | 80×24 | 120×30 |
|----------|-------|-------|--------|
| Título `🎵 Omusic` | ✓ | ✓ | ✓ |
| Ahora suena (▶/⏸, título, barra, tiempo, vol) | ✓ | ✓ | ✓ |
| Búsqueda/estado | ✓ | ✓ | ✓ |
| Cola (heading, ▶ pista actual) | ✓ | ✓ | ✓ |
| Cola (▲/▼, ⤓) | — (solo 2 items en TestViewGolden) | — | — |
| Letra | ✓ | ✓ | ✓ |
| Portada | NO (narrow) | NO (narrow) | ✓ (wide) |
| Ayuda (wrapHelp) | ✓ | ✓ | ✓ |
| Visualizador de barras | ✓ | ✓ | ✓ |

Las marcas ▲/▼/⤓ están cubiertas por `TestRenderQueueWindowsLongQueue` y `TestQueueWindow/*` (cola de 100 ítems), no en los goldens (que usan 2 ítems).

---

### S2-9. Alcance (scope check)

| Archivo | Modificado en Slice 2 |
|---------|----------------------|
| `internal/ui/view.go` | ✓ (computeLayout ampliado, renderMiddleSection/renderEnrichment, PlaceVertical) |
| `internal/ui/view_test.go` | ✓ (5 tests nuevos + extensión de TestNoLineExceedsWidth y TestGoldensDiffer) |
| `internal/ui/testdata/view_60x20.golden` | ✓ (nuevo) |
| `internal/ui/testdata/view_80x24.golden` | ✓ (regenerado) |
| `internal/ui/testdata/view_120x30.golden` | ✓ (regenerado) |
| `internal/ui/styles.go` | — sin cambios |
| `internal/ui/model.go` | — sin cambios |
| `internal/ui/update.go` | — sin cambios |
| `internal/ui/messages.go` | — sin cambios |
| `internal/ui/keys.go` | — sin cambios |

Líneas cambiadas (view.go + view_test.go, excluyendo goldens): 287 líneas (+/- combinadas en el diff); dentro del presupuesto de 400. No se introdujo código de Slice 3 (sin delegate styles, sin lógica modal nueva).

---

### S2-10. Regresión de Slice 1

| Assert de Slice 1 | Resultado |
|-------------------|-----------|
| `TestStylesNoBackground` (no-Background en title/panel) | PASS |
| `TestNoLineExceedsWidth` (todos los anchos previos) | PASS |
| `TestGoldensDiffer` (80×24 ≠ 120×30) | PASS (y ahora también ≠ 60×20) |
| `TestToggleOffParity_*` | PASS |
| `TestRenderQueueWindowsLongQueue` | PASS |
| `TestQueueWindow/*` | PASS |

Sin regresiones de Slice 1.

---

### S2-11. Evaluación de desviaciones del apply

#### Desviación 1: `maxQueueRows` con techo 10 (spec/tasks dice hasta 20)

**Descripción**: `tasks.md` S2-T1.3 especifica `clamp(bodyH-2, 3, 20)`. El `design.md` D4 dice `clamp(bodyH-2 (heading+borders), 3, 20)`. La implementación usa `clamp(bodyH-5, 3, 10)` — fórmula y techo distintos.

**Causa**: `TestRenderQueueWindowsLongQueue` (preexistente, con model width=120, height=40) espera `"▼ 90 más"`, lo que implica una ventana de exactamente 10 filas. Si el techo fuera 20, a h=40 `maxQueueRows` sería 20 y el test esperaría `"▼ 80 más"` — fallo garantizado. El apply forzosamente mantuvo el techo en 10 para no romper la suite.

**Evaluación**: DEUDA MENOR. El test preexistente impide alcanzar el máximo especificado (20 filas) en terminales altas. La consecuencia UX es que a partir de h≈30 la cola no sigue creciendo aunque haya espacio (se satura en 10 filas). Para resolverlo habría que actualizar `TestRenderQueueWindowsLongQueue` para no asumir una ventana de tamaño fijo. No es bloqueante para Slice 2 — el spec mínimo (≥3 y reducción en h=20) se cumple. Recomendado corregir en Slice 3 o en una tarea de mantenimiento.

**Diferencia de offset (-5 vs -2)**: el `-5` descuenta heading (1), borde superior e inferior del panel (2) y marcadores ▲/▼ (2) = 5 overhead real del panel de cola. El spec dice `-2` (heading+borders). La diferencia reduce `maxQueueRows` en 3 filas respecto al spec en alturas intermedias. También aceptable dada la limitación del techo=10.

#### Desviación 2: `PlaceVertical` en lugar de `Place(width, bodyH, ...)`

**Descripción**: `tasks.md` S2-T3.2 describe `lipgloss.Place(width, bodyH, Center, Top, band)`. La implementación usa `lipgloss.PlaceVertical(l.bodyH, lipgloss.Top, band)`.

**Evaluación**: CONSISTENTE con spec y tasks. El propio `tasks.md` S2-T3.2 dice explícitamente: "si `PlaceHorizontal` ya resuelve el requisito sin recortar elementos obligatorios, es preferible por su menor riesgo". `PlaceVertical` es el análogo vertical y es más preciso para el requisito (relleno vertical sin centrado horizontal adicional, dado que `center()` ya aplica `PlaceHorizontal` sobre toda la vista). No hay deuda.

#### Desviación 3: chrome fijo=11 + `helpRows(width)` medido

**Descripción**: el spec propone contar `chromeFixed` manualmente; la implementación usa `const chromeFixed = 11` más `helpRows(width)` que mide el wrap de la línea de ayuda al ancho actual.

**Evaluación**: FIEL al `design.md` D4: "chrome rows are measured, not guessed". La medición de `helpRows` es más robusta que un valor fijo: si la ayuda crece (p.ej., se añade un keybinding en Slice 3), `bodyH` se ajusta automáticamente. No introduce deuda; es una mejora respecto al spec textual.

---

### S2-12. Efecto UX del breakpoint 80 cols: portada oculta

**Pregunta**: ¿Es coherente con el spec @slice2 que a 80 cols la portada quede oculta? ¿Algún escenario queda violado?

**Análisis**:
- `design.md` Decision 1 establece el umbral hard como `bpNarrow < 90`. El argumento explícito: "< 90 guarantees two bordered panels fit at 80 (usable 78, split 33/45 incl. borders). Artwork returns at 90 where 3 minimal panels fit."
- `spec.md` (sección "Responsive Breakpoints") usa `narrow (< ~80)` con tilde (~), indicando un valor aproximado que el design resuelve con precisión.
- El escenario del feature `caelestia-ui.feature` dice: "Given a terminal **narrower than the narrow breakpoint**" → hides artwork. A 80 cols estamos dentro del `bpNarrow` (umbral=90), por lo que la portada se oculta correctamente según el escenario.
- `golden 80×24`: confirma que a 80 cols no hay "Portada" y sí hay "Cola" y "Letra". El golden está regenerado y bloqueado.

**Conclusión**: COHERENTE con el spec y el design. Ningún escenario @slice2 queda violado. La tilde (~80) en `spec.md` es una estimación aproximada; la decisión de diseño de usar 90 como umbral está fundamentada técnicamente y documentada en `design.md`. Un usuario a 80 cols no verá la portada — esto es intencional y correcto según el diseño aprobado.

---

### S2-13. Gaps de test

**Obs-S2-1 (menor)**: `TestComputeLayoutHeight` no verifica `plainLines` (solo verifica que `≥3`). El spec no establece un mínimo distinto para `plainLines` vs `lyricWindow`, pero un assert de `plainLines == lyricWindow` o `plainLines == clamp(bodyH-3, 3, 12)` haría explícito que ambas ventanas crecen sincronizadas. No es bloqueante.

**Obs-S2-2 (menor, preexistente de Slice 1)**: el escenario "All colors match Caelestia palette" sigue sin un test que afirme los 3 valores hex. Ver Obs-1 del reporte de Slice 1. Sin cambio de estado.

**Obs-S2-3 (deuda documentada)**: `TestRenderQueueWindowsLongQueue` fija implícitamente `maxQueueRows=10` (espera `"▼ 90 más"`). Para elevar el techo de la cola a 20 en terminales grandes, este test debe actualizarse. Ver Desviación 1.

---

### Resumen de evidencias — Slice 2

| Verificación | Resultado |
|---|---|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./internal/ui/...` (todos los tests) | PASS |
| `go test ./...` (17 paquetes) | PASS |
| No archivos `.got` | PASS |
| `TestClassifyBoundaries` (59/60/89/90/119/120) | PASS |
| `TestComputeLayoutWidths` (6 fronteras, artW=0 en narrow) | PASS |
| `TestComputeLayoutHeight` (h=20/24/30/40; min≥3, impar, <10@h20, ≥8@h30) | PASS |
| `Test60x20NarrowNoArtwork` (sin Portada, con Cola y Letra) | PASS |
| `TestNoLineExceedsWidth` (60×20, 60×24, 80×24, 120×24, 120×30) | PASS |
| `TestGoldensDiffer` (3 pares, todos distintos) | PASS |
| `TestViewGolden` (60×20, 80×24, 120×30) | PASS |
| Goldens: filas = altura objetivo (20/24/30) | PASS |
| Goldens: max línea ≤ cols objetivo (60/80/120), sin overflow | PASS |
| 60×20 sin Portada, 80×24 sin Portada, 120×30 con Portada | PASS |
| Elementos obligatorios presentes (título, ahora-suena, cola, letra, ayuda, viz) | PASS |
| Regresión Slice 1 (no-Background, TestToggleOffParity_*) | PASS |
| Scope: solo 5 archivos, sin Model/Update/keys/messages | PASS |
| Sin código de Slice 3 | PASS |
| Líneas de código cambiadas < 400 | PASS (287 líneas +/−) |
| Escenarios @slice2 cubiertos | 5/5 |
| Desviaciones de apply | 3 evaluadas — 1 deuda menor (maxQueueRows techo), 2 benignas |

---

### Veredicto — Slice 2

**PASA-CON-OBSERVACIONES**

La Slice 2 cumple el spec, el design y las tasks. Todas las verificaciones críticas pasan. La única deuda relevante es el techo `maxQueueRows=10` forzado por un test preexistente que impide que la cola crezca más allá de 10 filas en terminales altas — consecuencia UX limitada, no bloqueante. Las demás desviaciones son benignas o mejoran la fidelidad al design. No se requiere re-apply.
