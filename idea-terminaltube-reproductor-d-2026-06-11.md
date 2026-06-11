---
rawIdea: |
  # TerminalTube - Reproductor de Música en Terminal para Linux/Omarchy usando YouTube
  
  ## Objetivo
  
  Desarrollar un reproductor de música basado en terminal (TUI) compatible con Omarchy/Linux que permita buscar, reproducir y gestionar música utilizando YouTube como fuente principal de audio.
  
  El proyecto está orientado a uso personal, por lo que puede apoyarse en herramientas como `yt-dlp` para obtener las URLs de streaming y `mpv` como motor de reproducción.
  
  ---
  
  # Alcance Inicial (MVP)
  
  El primer objetivo es construir una aplicación que permita:
  
  * Buscar canciones en YouTube.
  * Mostrar resultados en una interfaz de terminal.
  * Reproducir audio directamente desde YouTube.
  * Gestionar una cola de reproducción.
  * Controlar reproducción mediante atajos de teclado.
  * Guardar historial local.
  
  No se busca una integración oficial con YouTube Music ni con YouTube Premium.
  
  ---
  
  # Arquitectura General
  
  ```text
  ┌─────────────────────────┐
  │      Terminal UI        │
  │ (BubbleTea / Textual)   │
  └───────────┬─────────────┘
              │
              ▼
  ┌─────────────────────────┐
  │     Search Service      │
  │  yt-dlp / Invidious     │
  └───────────┬─────────────┘
              │
              ▼
  ┌─────────────────────────┐
  │   Playback Controller   │
  └───────────┬─────────────┘
              │
              ▼
  ┌─────────────────────────┐
  │         MPV             │
  │  Streaming de audio     │
  └─────────────────────────┘
  ```
  
  ---
  
  # Tecnologías Propuestas
  
  ## Opción A: Go (Recomendada)
  
  ### Ventajas
  
  * Binarios únicos.
  * Muy bajo consumo de memoria.
  * Excelente experiencia para herramientas CLI.
  * Fácil distribución.
  
  ### Librerías
  
  #### UI
  
  * Bubble Tea
  * Lip Gloss
  
  #### Reproducción
  
  * mpv
  
  #### Extracción de audio
  
  * yt-dlp
  
  #### Configuración
  
  * Viper
  
  #### Base de datos local
  
  * SQLite
  
  ---
  
  ## Opción B: Rust
  
  ### Ventajas
  
  * Máximo rendimiento.
  * Consumo mínimo de recursos.
  
  ### Desventajas
  
  * Mayor complejidad.
  * Curva de aprendizaje más pronunciada.
  
  ---
  
  ## Opción C: Python
  
  ### Ventajas
  
  * Desarrollo extremadamente rápido.
  * Ecosistema maduro.
  
  ### Desventajas
  
  * Distribución más complicada.
  * Mayor consumo de recursos.
  
  ---
  
  # Flujo de Reproducción
  
  Cuando el usuario selecciona una canción:
  
  ```bash
  yt-dlp -f ba --get-url "https://youtube.com/watch?v=VIDEO_ID"
  ```
  
  Obtiene una URL temporal de audio.
  
  Posteriormente:
  
  ```bash
  mpv URL_AUDIO
  ```
  
  MPV se encarga de:
  
  * Buffering
  * Decodificación
  * Control de volumen
  * Streaming
  
  La aplicación únicamente controla el proceso.
  
  ---
  
  # Funcionalidades MVP
  
  ## Búsqueda
  
  ```text
  Buscar:
  > Linkin Park Numb
  ```
  
  Resultados:
  
  ```text
  1. Numb - Linkin Park
  2. Numb (Live)
  3. Numb (Remastered)
  ```
  
  ---
  
  ## Cola
  
  ```text
  ▶ Numb
    In The End
    Faint
    Crawling
  ```
  
  ---
  
  ## Controles
  
  ```text
  Espacio  Play/Pause
  n        Siguiente
  p        Anterior
  +        Volumen +
  -        Volumen -
  q        Salir
  /        Buscar
  ```
  
  ---
  
  # Funcionalidades Futuras
  
  ## Playlists
  
  Guardar listas locales:
  
  ```json
  {
    "name": "Trabajo",
    "songs": [
      "VIDEO_ID_1",
      "VIDEO_ID_2"
    ]
  }
  ```
  
  ---
  
  ## Favoritos
  
  ```text
  ♥ Canciones favoritas
  ```
  
  ---
  
  ## Historial
  
  ```text
  Últimas reproducciones
  ```
  
  ---
  
  ## Caché local
  
  Guardar metadatos:
  
  ```text
  Título
  Artista
  Duración
  Thumbnail
  Video ID
  ```
  
  Evita búsquedas repetidas.
  
  ---
  
  # Integración con YouTube Premium
  
  Actualmente no existe una API pública oficial que permita crear un cliente de streaming musical equivalente a Spotify Connect.
  
  Por ello el proyecto utilizará:
  
  ```text
  YouTube
       ↓
  yt-dlp
       ↓
  URL de audio temporal
       ↓
  mpv
       ↓
  Reproducción
  ```
  
  Este enfoque es adecuado para un proyecto personal, aunque depende de que YouTube no modifique los mecanismos que utiliza yt-dlp.
  
  ---
  
  # Roadmap
  
  ## Fase 1
  
  * Buscar canciones
  * Reproducir audio
  * Cola de reproducción
  * Atajos de teclado
  
  Tiempo estimado:
  
  1 fin de semana
  
  ---
  
  ## Fase 2
  
  * Playlists
  * Favoritos
  * Historial
  * Persistencia SQLite
  
  Tiempo estimado:
  
  1 semana
  
  ---
  
  ## Fase 3
  
  * Descarga temporal para caché
  * Letras
  * Portadas
  * Integración con Discord Rich Presence
  
  Tiempo estimado:
  
  2 semanas
  
  ---
  
  # Stack Final Recomendado
  
  ```text
  Lenguaje:      Go
  UI:            Bubble Tea
  Player:        MPV
  Extractor:     yt-dlp
  DB Local:      SQLite
  Config:        Viper
  Logs:          Zap
  ```
  
  Este stack ofrece el mejor equilibrio entre rendimiento, simplicidad y facilidad de distribución para una aplicación TUI moderna en Linux/Omarchy.
answers:
  audience: ""
  problem: ""
  unique: ""
  mvp: ""
  open: ""
status: someday
createdAt: "2026-06-11T00:34:52.366Z"
---

# Idea: # TerminalTube - Reproductor de Música en Terminal para Linu...
Captured: 2026-06-11T00:34:52.366Z

## Raw Idea
# TerminalTube - Reproductor de Música en Terminal para Linux/Omarchy usando YouTube

## Objetivo

Desarrollar un reproductor de música basado en terminal (TUI) compatible con Omarchy/Linux que permita buscar, reproducir y gestionar música utilizando YouTube como fuente principal de audio.

El proyecto está orientado a uso personal, por lo que puede apoyarse en herramientas como `yt-dlp` para obtener las URLs de streaming y `mpv` como motor de reproducción.

---

# Alcance Inicial (MVP)

El primer objetivo es construir una aplicación que permita:

* Buscar canciones en YouTube.
* Mostrar resultados en una interfaz de terminal.
* Reproducir audio directamente desde YouTube.
* Gestionar una cola de reproducción.
* Controlar reproducción mediante atajos de teclado.
* Guardar historial local.

No se busca una integración oficial con YouTube Music ni con YouTube Premium.

---

# Arquitectura General

```text
┌─────────────────────────┐
│      Terminal UI        │
│ (BubbleTea / Textual)   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│     Search Service      │
│  yt-dlp / Invidious     │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│   Playback Controller   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│         MPV             │
│  Streaming de audio     │
└─────────────────────────┘
```

---

# Tecnologías Propuestas

## Opción A: Go (Recomendada)

### Ventajas

* Binarios únicos.
* Muy bajo consumo de memoria.
* Excelente experiencia para herramientas CLI.
* Fácil distribución.

### Librerías

#### UI

* Bubble Tea
* Lip Gloss

#### Reproducción

* mpv

#### Extracción de audio

* yt-dlp

#### Configuración

* Viper

#### Base de datos local

* SQLite

---

## Opción B: Rust

### Ventajas

* Máximo rendimiento.
* Consumo mínimo de recursos.

### Desventajas

* Mayor complejidad.
* Curva de aprendizaje más pronunciada.

---

## Opción C: Python

### Ventajas

* Desarrollo extremadamente rápido.
* Ecosistema maduro.

### Desventajas

* Distribución más complicada.
* Mayor consumo de recursos.

---

# Flujo de Reproducción

Cuando el usuario selecciona una canción:

```bash
yt-dlp -f ba --get-url "https://youtube.com/watch?v=VIDEO_ID"
```

Obtiene una URL temporal de audio.

Posteriormente:

```bash
mpv URL_AUDIO
```

MPV se encarga de:

* Buffering
* Decodificación
* Control de volumen
* Streaming

La aplicación únicamente controla el proceso.

---

# Funcionalidades MVP

## Búsqueda

```text
Buscar:
> Linkin Park Numb
```

Resultados:

```text
1. Numb - Linkin Park
2. Numb (Live)
3. Numb (Remastered)
```

---

## Cola

```text
▶ Numb
  In The End
  Faint
  Crawling
```

---

## Controles

```text
Espacio  Play/Pause
n        Siguiente
p        Anterior
+        Volumen +
-        Volumen -
q        Salir
/        Buscar
```

---

# Funcionalidades Futuras

## Playlists

Guardar listas locales:

```json
{
  "name": "Trabajo",
  "songs": [
    "VIDEO_ID_1",
    "VIDEO_ID_2"
  ]
}
```

---

## Favoritos

```text
♥ Canciones favoritas
```

---

## Historial

```text
Últimas reproducciones
```

---

## Caché local

Guardar metadatos:

```text
Título
Artista
Duración
Thumbnail
Video ID
```

Evita búsquedas repetidas.

---

# Integración con YouTube Premium

Actualmente no existe una API pública oficial que permita crear un cliente de streaming musical equivalente a Spotify Connect.

Por ello el proyecto utilizará:

```text
YouTube
     ↓
yt-dlp
     ↓
URL de audio temporal
     ↓
mpv
     ↓
Reproducción
```

Este enfoque es adecuado para un proyecto personal, aunque depende de que YouTube no modifique los mecanismos que utiliza yt-dlp.

---

# Roadmap

## Fase 1

* Buscar canciones
* Reproducir audio
* Cola de reproducción
* Atajos de teclado

Tiempo estimado:

1 fin de semana

---

## Fase 2

* Playlists
* Favoritos
* Historial
* Persistencia SQLite

Tiempo estimado:

1 semana

---

## Fase 3

* Descarga temporal para caché
* Letras
* Portadas
* Integración con Discord Rich Presence

Tiempo estimado:

2 semanas

---

# Stack Final Recomendado

```text
Lenguaje:      Go
UI:            Bubble Tea
Player:        MPV
Extractor:     yt-dlp
DB Local:      SQLite
Config:        Viper
Logs:          Zap
```

Este stack ofrece el mejor equilibrio entre rendimiento, simplicidad y facilidad de distribución para una aplicación TUI moderna en Linux/Omarchy.

## Who is this for?


## What problem does it solve?


## What makes it unique?


## How would you build it (MVP)?


## What are the open questions?


---
*Generated by Idea Log*
