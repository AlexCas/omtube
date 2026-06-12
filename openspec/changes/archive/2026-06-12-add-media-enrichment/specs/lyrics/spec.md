# Lyrics Specification

## Purpose

Obtener la letra de la pista en reproducción desde una API comunitaria sin auth y
mostrarla; si hay letra sincronizada (.lrc), resaltar la línea según la posición de
reproducción.

## Requirements

### Requirement: Fetch Lyrics

The system MUST fetch lyrics for the current track from a community API (by
title/artist), preferring synced (.lrc) lyrics and falling back to plain text. Fetching
MUST be controllable by a config toggle, and results SHOULD be cached.

#### Scenario: Synced lyrics found

- GIVEN el toggle de letras está activo y existe letra sincronizada
- WHEN suena una pista
- THEN se obtiene y parsea el .lrc con marcas de tiempo

#### Scenario: Plain lyrics fallback

- GIVEN no hay letra sincronizada pero sí texto plano
- WHEN suena la pista
- THEN se muestra la letra sin sincronización

### Requirement: Lyrics Unavailable

The system MUST handle API failure or no-match without crashing, indicating that no
lyrics are available.

#### Scenario: No match

- GIVEN la API no encuentra letra para la pista
- WHEN se solicita la letra
- THEN se indica "sin letra" y la reproducción continúa normal

#### Scenario: API down

- GIVEN la API de letras no responde o devuelve error
- WHEN se solicita la letra
- THEN se trata como "sin letra" sin bloquear la UI

### Requirement: Synced Line Highlight

The system MUST highlight the lyric line matching the current playback position when
synced lyrics are available, advancing as playback progresses.

#### Scenario: Highlight advances with playback

- GIVEN hay letra sincronizada cargada
- WHEN la posición de reproducción avanza
- THEN la línea resaltada cambia a la correspondiente al tiempo actual

#### Scenario: Seek updates highlight

- GIVEN hay letra sincronizada y el usuario salta de posición
- WHEN cambia la posición
- THEN la línea resaltada salta a la correspondiente al nuevo tiempo
