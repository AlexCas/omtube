# Discord Rich Presence Specification

## Purpose

Publicar la pista en reproducción como presencia "escuchando" en Discord mediante su
IPC local, como feature opcional que falla en silencio si Discord no está disponible.

## Requirements

### Requirement: Presence Connection

The system MUST connect to the local Discord IPC when the feature is enabled, and MUST
treat connection failure as a silent no-op (the app keeps working). Presence MUST be
controllable by a config toggle.

#### Scenario: Discord running

- GIVEN el toggle de presencia está activo y Discord está corriendo
- WHEN la app inicia la presencia
- THEN se establece la conexión IPC con Discord

#### Scenario: Discord not running

- GIVEN el toggle está activo pero Discord no está presente/abierto
- WHEN la app intenta conectarse
- THEN falla en silencio y la app continúa sin presencia

#### Scenario: Presence disabled

- GIVEN el toggle de presencia está desactivado
- WHEN se reproduce una pista
- THEN no se intenta ninguna conexión IPC

### Requirement: Publish Now Playing

The system MUST publish the current track (title/artist) as Discord activity and MUST
update it on track change, clearing presence when playback stops or the app exits.

#### Scenario: Publish on play

- GIVEN la presencia está conectada
- WHEN empieza a sonar una pista
- THEN Discord muestra "escuchando: <título>"

#### Scenario: Update on track change

- GIVEN una presencia publicada para la pista actual
- WHEN avanza a otra pista
- THEN la actividad de Discord se actualiza a la nueva pista

#### Scenario: Clear on stop or exit

- GIVEN una presencia activa
- WHEN la reproducción se detiene o la app termina
- THEN la actividad de Discord se limpia
