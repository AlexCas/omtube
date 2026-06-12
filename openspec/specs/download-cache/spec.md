# Download Cache Specification

## Purpose

Descargar y guardar el audio en una caché local (directorio XDG cache + índice),
para reproducir el archivo local en repeticiones sin re-resolver ni re-descargar, con
expiración por tamaño/antigüedad.

## Requirements

### Requirement: Local Audio Cache

The system MUST cache downloaded audio under the XDG cache directory and MUST record an
index entry (track id, file path, size, timestamp) so a cached track can be located
later. Caching MUST be controllable by a config toggle.

#### Scenario: Cache on first play

- GIVEN el toggle de caché está activo y la pista no está cacheada
- WHEN la pista se reproduce
- THEN el audio se descarga al directorio de caché y se registra en el índice

#### Scenario: Cache disabled

- GIVEN el toggle de caché está desactivado
- WHEN la pista se reproduce
- THEN no se escribe ningún archivo de caché y se reproduce por streaming

### Requirement: Cache Lookup Priority

The system MUST check the cache index before resolving/streaming a track and MUST serve
the local file when a valid cached entry exists.

#### Scenario: Serve cached track

- GIVEN una pista ya cacheada y válida
- WHEN el usuario la reproduce de nuevo
- THEN se reproduce el archivo local sin volver a descargar ni resolver

#### Scenario: Cached file missing or corrupt

- GIVEN existe una entrada de índice pero el archivo falta o está corrupto
- WHEN se intenta reproducir esa pista
- THEN la entrada se invalida y la pista se vuelve a descargar/streamear

### Requirement: Cache Eviction

The system MUST enforce a configurable maximum cache size and/or age, evicting the
oldest entries when a limit is exceeded.

#### Scenario: Evict on size limit

- GIVEN la caché alcanza el límite de tamaño configurado
- WHEN se añade una nueva descarga
- THEN se eliminan las entradas más antiguas hasta quedar bajo el límite
- AND el índice se actualiza para reflejar las eliminaciones

#### Scenario: Clear cache

- GIVEN existen archivos cacheados
- WHEN el usuario borra el directorio de caché
- THEN la app sigue funcionando y trata todas las pistas como no cacheadas
