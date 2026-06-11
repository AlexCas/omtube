# Delta for playback-queue

## ADDED Requirements

### Requirement: Enqueue Tracks

The system MUST append tracks; the first into an empty queue becomes current.

#### Scenario: Enqueue into empty queue

- GIVEN cola vacía
- WHEN se encola una pista
- THEN pasa a ser la actual

### Requirement: Advance and Rewind

The system MUST support next/previous; advancing past the last track stops (no wrap).

#### Scenario: Next at end

- GIVEN la actual es la última
- WHEN se solicita siguiente
- THEN no hay actual reproducible y la reproducción se detiene

#### Scenario: Previous track

- GIVEN no es la primera
- WHEN se solicita anterior
- THEN la actual retrocede una posición
