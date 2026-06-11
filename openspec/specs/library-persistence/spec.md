# Library Persistence Specification

## Purpose

Capa de almacenamiento local en SQLite que respalda biblioteca, playlists, favoritos e
historial. Provee un esquema versionado con migraciones y conserva el binario único
(driver sin cgo).

## Requirements

### Requirement: Single-File Local Database

The system MUST store library data in a single local SQLite database file under the XDG
data dir (`~/.local/share/terminaltube/library.db`). The driver MUST be pure Go (no cgo)
to preserve the single-binary distribution.

#### Scenario: Create database on first run

- GIVEN no existe el archivo de base de datos
- WHEN la aplicación arranca
- THEN se crea el archivo en la ruta XDG y queda lista para usarse

#### Scenario: Reuse existing database

- GIVEN ya existe el archivo de base de datos
- WHEN la aplicación arranca
- THEN abre el archivo existente sin perder datos

### Requirement: Versioned Schema with Migrations

The system MUST track a schema version and apply pending migrations in order on startup.
Migrations MUST be idempotent: running an already-applied schema MUST NOT re-run
migrations or alter data.

#### Scenario: Initialize fresh schema

- GIVEN una base de datos sin esquema
- WHEN la aplicación arranca
- THEN se aplican todas las migraciones y la versión queda en la más reciente

#### Scenario: Already up to date

- GIVEN la base de datos ya está en la versión más reciente
- WHEN la aplicación arranca
- THEN no se aplica ninguna migración y los datos no cambian

#### Scenario: Apply pending migration

- GIVEN la base de datos está en una versión anterior
- WHEN la aplicación arranca
- THEN se aplican solo las migraciones pendientes en orden hasta la versión actual

### Requirement: Track Identity

The system MUST identify tracks by their video id as the natural key, storing at least
title and uploader, so playlists, favorites, and history reference the same track record.

#### Scenario: Reuse track across features

- GIVEN una pista ya registrada por su video id
- WHEN se referencia desde una playlist, favoritos o historial
- THEN se reutiliza el mismo registro de pista sin duplicarlo

### Requirement: Entity Repositories with CRUD

The system MUST expose repository operations to create, read, update, and delete library
entities (tracks, playlists, playlist membership, favorites, history). Operations MUST
return errors instead of panicking on failure.

#### Scenario: CRUD round-trip persists

- GIVEN un repositorio de una entidad
- WHEN se crea, se lee, se actualiza y se borra un registro
- THEN cada operación refleja el estado esperado y persiste en disco

#### Scenario: Read missing record

- GIVEN un identificador que no existe
- WHEN se solicita leerlo
- THEN se devuelve un resultado vacío o un error claro, sin pánico
