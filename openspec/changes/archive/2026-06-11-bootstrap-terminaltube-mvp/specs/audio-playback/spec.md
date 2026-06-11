# Delta for audio-playback

## ADDED Requirements

### Requirement: Single mpv via IPC

The system MUST launch one idle `mpv` with an IPC socket and control it via JSON,
reusing the process across tracks.

#### Scenario: Start player

- GIVEN mpv en PATH
- WHEN la app arranca
- THEN se lanza mpv idle con input-ipc-server y la app se conecta

### Requirement: Load and Transport Control

The system MUST load by YouTube id and control pause/volume via loadfile/set_property.

#### Scenario: Toggle pause and volume

- GIVEN una pista en reproducción
- WHEN el usuario alterna pausa o ajusta volumen
- THEN mpv aplica el cambio (volumen en rango 0–130)

### Requirement: Progress and End Events

The system MUST expose position/duration and emit an end event over a channel.

#### Scenario: Track ends

- GIVEN una pista termina
- WHEN mpv emite end-file
- THEN se emite evento de fin para avanzar la cola
