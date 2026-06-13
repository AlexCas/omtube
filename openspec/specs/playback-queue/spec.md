# Playback Queue Specification

## Purpose

Gestionar el orden de reproducción: pista actual, encolado, avance y retroceso, con
auto-avance al terminar una pista.

## Requirements

### Requirement: Enqueue Tracks

The system MUST allow appending one or more tracks to the queue. The first track
appended to an empty queue becomes the current track.

#### Scenario: Enqueue into empty queue

- GIVEN la cola está vacía
- WHEN se encola una pista
- THEN esa pista pasa a ser la actual

#### Scenario: Append to non-empty queue

- GIVEN la cola tiene pistas
- WHEN se encola otra
- THEN se añade al final sin cambiar la actual

### Requirement: Advance and Rewind

The system MUST support moving to next and previous tracks. Advancing past the last
track stops at the end (no wrap by default).

#### Scenario: Next track

- GIVEN hay una pista siguiente
- WHEN se solicita siguiente
- THEN la actual avanza a la siguiente

#### Scenario: Next at end

- GIVEN la actual es la última
- WHEN se solicita siguiente
- THEN no hay pista actual reproducible y la reproducción se detiene

#### Scenario: Previous track

- GIVEN no es la primera pista
- WHEN se solicita anterior
- THEN la actual retrocede una posición

### Requirement: Clear Queue

The system MUST allow clearing the entire queue. Clearing MUST remove all tracks, leave
no current track, and stop playback of the current track.

#### Scenario: Clear a populated queue

- GIVEN la cola tiene pistas y una se está reproduciendo
- WHEN el usuario limpia la cola
- THEN la cola queda vacía, no hay pista actual y la reproducción se detiene

#### Scenario: Clear an empty queue

- GIVEN la cola ya está vacía
- WHEN el usuario limpia la cola
- THEN no hay cambios y no ocurre ningún error
