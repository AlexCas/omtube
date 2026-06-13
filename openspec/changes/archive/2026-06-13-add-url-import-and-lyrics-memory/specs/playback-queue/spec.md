# Delta for playback-queue

## ADDED Requirements

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
