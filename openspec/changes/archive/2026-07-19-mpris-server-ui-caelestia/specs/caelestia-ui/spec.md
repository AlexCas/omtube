# Caelestia UI Specification

## Purpose

Rediseñar la interfaz TUI de Omusic con bordes redondeados, secciones
definidas, y una paleta de colores cohesiva estilo Caelestia, manteniendo
todos los atajos de teclado existentes.

## Palette

| Token | Hex |
|-------|-----|
| primary (deep blue) | `#1a1a2e` |
| accent (gold) | `#e0aaff` |
| muted text | `#a0a0a0` |
| highlight (teal) | `#00f5d4` |

Applied via `lipgloss` styles in `styles.go`.

## Requirements

### Requirement: Rounded Borders

All panels (queue, lyrics, artwork, now-playing bar) MUST use
`lipgloss.RoundedBorder()` with consistent padding. Borders MUST render
correctly at terminal widths ≥80 and heights ≥24.

#### Scenario: TUI renders with rounded borders at 80×24

### Requirement: Defined Sections

Each functional area MUST be visually separated with consistent border
styling and padding. Sections MUST include: now-playing bar (top), queue
panel, lyrics panel, artwork panel. Visual hierarchy MUST be clear.

#### Scenario: Queue panel shows tracks with correct highlighting

#### Scenario: Lyrics panel displays synced lyrics with active line highlighted

#### Scenario: Artwork panel renders when available

### Requirement: Caelestia Palette

The system MUST apply the Caelestia palette consistently: primary
`#1a1a2e` for backgrounds, accent `#e0aaff` for active elements, muted
`#a0a0a0` for secondary text, highlight `#00f5d4` for selection. No
legacy pink/blue/green colors MUST remain.

#### Scenario: All colors match Caelestia palette after redesign

### Requirement: Now Playing Bar

The system MUST render a now-playing bar with playback state, track title,
progress bar, time, and volume displayed cohesively in a single bar.

#### Scenario: Now playing bar shows progress and controls

### Requirement: Queue Panel

The queue panel MUST display tracks with a sliding window, highlight the
current track, and show a cache indicator per track. It MUST scroll to
keep the current track visible.

#### Scenario: Queue panel shows tracks with sliding window and cache indicator

### Requirement: Layout Resilience

At 80×24, the TUI MUST not overflow or break. Content longer than the
available width MUST truncate gracefully with ellipsis. At larger sizes,
panels MUST expand proportionally.

#### Scenario: Small terminal (80×24) does not overflow or break layout

### Requirement: Keyboard Shortcut Preservation

All existing keyboard shortcuts (space, `n`, `p`, `+`/`-`, `/`, `q`,
library/favorites actions) MUST remain functional after the redesign.
No shortcut mapping MUST change.

#### Scenario: All existing keyboard shortcuts still work after redesign
