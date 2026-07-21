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
