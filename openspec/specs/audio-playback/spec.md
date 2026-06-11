# Audio Playback Specification

## Purpose

Reproducir audio de YouTube controlando un proceso `mpv` único mediante IPC sobre
socket Unix, exponiendo control de transporte y estado de reproducción a la app.

## Requirements

### Requirement: Single mpv via IPC

The system MUST launch one `mpv` instance in idle mode with an IPC server socket and
control it with JSON commands, reusing the same process across tracks.

#### Scenario: Start player

- GIVEN mpv está disponible en PATH
- WHEN la app arranca
- THEN se lanza `mpv --idle --no-video --input-ipc-server=<sock>`
- AND la app se conecta al socket

#### Scenario: mpv missing

- GIVEN mpv no está en PATH
- WHEN la app arranca
- THEN se reporta un error legible y la app no intenta reproducir

### Requirement: Load and Transport Control

The system MUST load a track by YouTube id and control pause and volume via
`loadfile` and `set_property`.

#### Scenario: Play a track

- GIVEN el reproductor está activo
- WHEN se carga el id de un video
- THEN mpv reproduce su audio (resuelto por su hook yt-dlp, formato bestaudio)

#### Scenario: Toggle pause and volume

- GIVEN una pista en reproducción
- WHEN el usuario alterna pausa o ajusta volumen
- THEN mpv aplica el cambio vía set_property
- AND el volumen se mantiene en el rango 0–130

### Requirement: Progress and End Events

The system MUST expose current position/duration and emit an event when a track
ends, over an event channel.

#### Scenario: Track ends

- GIVEN una pista llega a su fin
- WHEN mpv emite `end-file`
- THEN se emite un evento de fin por el canal para que la cola avance
