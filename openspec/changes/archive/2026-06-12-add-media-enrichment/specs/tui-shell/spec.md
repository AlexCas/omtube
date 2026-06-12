# Delta for tui-shell

## ADDED Requirements

### Requirement: Lyrics Panel

The system MUST provide a panel that displays the current track's lyrics and, when
synced lyrics are available, highlights the line matching playback position. The panel
MUST show a clear "sin letra" state when none are available, and MUST NOT block the UI.

#### Scenario: Show lyrics

- GIVEN hay letra disponible para la pista actual
- WHEN el panel de letra está visible
- THEN se muestra la letra; si es sincronizada, se resalta la línea actual

#### Scenario: No lyrics state

- GIVEN no hay letra disponible para la pista
- WHEN el panel de letra está visible
- THEN se muestra un estado "sin letra" sin afectar el resto de la UI

### Requirement: Artwork Panel

The system MUST provide a panel that displays the current track's artwork using the
terminal's image capability, degrading gracefully (chafa o sin-portada) when images are
unsupported.

#### Scenario: Show artwork

- GIVEN el terminal soporta imágenes y hay portada disponible
- WHEN el panel de portada está visible
- THEN se renderiza la portada de la pista actual

#### Scenario: Degrade without image support

- GIVEN el terminal no soporta imágenes
- WHEN el panel de portada está visible
- THEN se muestra una degradación (chafa/placeholder) sin error

### Requirement: Cache Indicator

The system MUST indicate, per track in results/queue, whether it is available in the
local cache.

#### Scenario: Show cached status

- GIVEN una pista está cacheada localmente
- WHEN se muestra en resultados o en la cola
- THEN la UI muestra un indicador de "cacheada" para esa pista
