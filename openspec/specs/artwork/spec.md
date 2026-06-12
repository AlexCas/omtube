# Artwork Specification

## Purpose

Mostrar la portada/thumbnail de la pista en el terminal usando el protocolo gráfico
disponible (kitty graphics o sixel), con degradación elegante a `chafa` (ASCII/blocks)
o a sin-portada cuando el terminal no soporta imágenes.

## Requirements

### Requirement: Terminal Graphics Detection

The system MUST detect the terminal's image capability and select a render path: kitty
graphics, sixel, `chafa` fallback, or no-image. Artwork MUST be controllable by a config
toggle.

#### Scenario: Capable terminal

- GIVEN el terminal soporta kitty graphics o sixel
- WHEN se muestra la portada
- THEN se renderiza la imagen con el protocolo soportado

#### Scenario: Unsupported terminal

- GIVEN el terminal no soporta protocolos de imagen
- WHEN se intenta mostrar la portada
- THEN se degrada a `chafa` si está disponible, o a sin-portada, sin error

### Requirement: Render Current Track Artwork

The system MUST fetch and display the artwork for the current track and MUST update it
when the track changes.

#### Scenario: Show artwork on play

- GIVEN el toggle de portada está activo y el terminal es compatible
- WHEN empieza una pista con thumbnail disponible
- THEN se muestra su portada en el panel correspondiente

#### Scenario: Update on track change

- GIVEN se muestra la portada de la pista actual
- WHEN avanza a la siguiente pista
- THEN la portada se actualiza a la de la nueva pista

#### Scenario: Artwork unavailable

- GIVEN una pista sin thumbnail o cuya descarga falla
- WHEN se intenta mostrar la portada
- THEN el panel queda vacío/placeholder sin interrumpir la reproducción
