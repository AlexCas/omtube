# Delta for tui-shell

## ADDED Requirements

### Requirement: Search Input Mode

The system MUST provide a `/`-toggled search mode that captures text and runs a
search on submit.

#### Scenario: Enter and submit search

- GIVEN modo normal
- WHEN el usuario pulsa `/`, escribe y envía
- THEN se ejecuta la búsqueda y se muestran resultados

### Requirement: Results and Queue Panels

The system MUST show results and queue, highlighting the current track.

#### Scenario: Enqueue from results

- GIVEN resultados visibles
- WHEN se selecciona uno
- THEN se añade a la cola (y reproduce si estaba vacía)

### Requirement: Keyboard Controls

The system MUST bind espacio (play/pausa), `n`, `p`, `+`/`-`, `/`, `q`.

#### Scenario: Quit cleanly

- GIVEN app corriendo
- WHEN pulsa `q`
- THEN mpv se detiene, el socket se cierra y la app termina sin errores

### Requirement: Non-blocking UI

The system MUST run search/playback resolution asynchronously without freezing.

#### Scenario: Search does not block

- GIVEN una búsqueda en curso
- WHEN yt-dlp tarda
- THEN la UI responde y muestra estado de carga
